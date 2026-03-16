// Package catalog provides a local catalog cache for the edge service.
// Items are synced from the cloud catalog service periodically.
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Item struct {
	ItemID   string `json:"item_id"`
	VenueID  string `json:"venue_id"`
	Name     string `json:"name"`
	PriceNC  int    `json:"price_nc"`
	IsActive bool   `json:"is_active"`
	Category string `json:"category,omitempty"`
}

type Cache struct {
	db         *sql.DB
	catalogURL string
	venueID    string
}

func New(db *sql.DB, catalogURL, venueID string) *Cache {
	return &Cache{db: db, catalogURL: catalogURL, venueID: venueID}
}

// ListItems returns active catalog items for the venue.
func (c *Cache) ListItems(ctx context.Context) ([]*Item, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT item_id, venue_id, name, price_nc, is_active, COALESCE(category,'')
		FROM catalog_items WHERE venue_id=? AND is_active=1
		ORDER BY name
	`, c.venueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*Item
	for rows.Next() {
		item := &Item{}
		var isActive int
		if err := rows.Scan(&item.ItemID, &item.VenueID, &item.Name, &item.PriceNC,
			&isActive, &item.Category); err != nil {
			return nil, err
		}
		item.IsActive = isActive == 1
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetItem returns a catalog item by ID.
func (c *Cache) GetItem(ctx context.Context, itemID string) (*Item, error) {
	item := &Item{}
	var isActive int
	err := c.db.QueryRowContext(ctx, `
		SELECT item_id, venue_id, name, price_nc, is_active, COALESCE(category,'')
		FROM catalog_items WHERE item_id=?
	`, itemID).Scan(&item.ItemID, &item.VenueID, &item.Name, &item.PriceNC, &isActive, &item.Category)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("item not found")
	}
	item.IsActive = isActive == 1
	return item, err
}

// Sync fetches fresh catalog data from the cloud catalog service.
func (c *Cache) Sync(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/venues/%s/items", c.catalogURL, c.venueID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Internal-Service", "edge")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("catalog service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("catalog returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Items []Item `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode catalog: %w", err)
	}

	return c.upsertItems(ctx, result.Items)
}

func (c *Cache) upsertItems(ctx context.Context, items []Item) error {
	now := time.Now()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, item := range items {
		isActive := 0
		if item.IsActive {
			isActive = 1
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_items (item_id, venue_id, name, price_nc, is_active, category, updated_at)
			VALUES (?,?,?,?,?,?,?)
			ON CONFLICT(item_id) DO UPDATE
			SET name=EXCLUDED.name, price_nc=EXCLUDED.price_nc,
			    is_active=EXCLUDED.is_active, category=EXCLUDED.category, updated_at=EXCLUDED.updated_at
		`, item.ItemID, item.VenueID, item.Name, item.PriceNC, isActive, item.Category, now)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// StartSyncLoop runs the catalog sync in a background goroutine.
func (c *Cache) StartSyncLoop(ctx context.Context, interval time.Duration) {
	go func() {
		// Initial sync
		if err := c.Sync(ctx); err != nil {
			slog.Warn("initial catalog sync failed", "err", err)
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := c.Sync(ctx); err != nil {
					slog.Warn("catalog sync failed", "err", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
