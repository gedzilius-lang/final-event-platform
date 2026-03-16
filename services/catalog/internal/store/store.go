package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type Venue struct {
	VenueID       string    `json:"venue_id"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	City          string    `json:"city"`
	Address       string    `json:"address,omitempty"`
	Capacity      int       `json:"capacity"`
	Timezone      string    `json:"timezone"`
	Theme         []byte    `json:"theme"`
	StripeAccount string    `json:"stripe_account,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
}

type CatalogItem struct {
	ItemID             string    `json:"item_id"`
	VenueID            string    `json:"venue_id"`
	Name               string    `json:"name"`
	Category           string    `json:"category"`
	PriceNC            int       `json:"price_nc"`
	Icon               string    `json:"icon,omitempty"`
	StockQty           *int      `json:"stock_qty,omitempty"`
	LowThreshold       int       `json:"low_threshold"`
	IsActive           bool      `json:"is_active"`
	DisplayOrder       int       `json:"display_order"`
	HappyHourPriceNC   *int      `json:"happy_hour_price_nc,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Event struct {
	EventID     string     `json:"event_id"`
	VenueID     string     `json:"venue_id"`
	Title       string     `json:"title"`
	StartsAt    time.Time  `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
	Description string     `json:"description,omitempty"`
	Genre       string     `json:"genre,omitempty"`
	ImageURL    string     `json:"image_url,omitempty"`
	IsPublic    bool       `json:"is_public"`
	CreatedAt   time.Time  `json:"created_at"`
}

type HappyHourRule struct {
	RuleID        string  `json:"rule_id"`
	VenueID       string  `json:"venue_id"`
	Name          string  `json:"name"`
	StartsAt      string  `json:"starts_at"` // HH:MM
	EndsAt        string  `json:"ends_at"`
	DaysOfWeek    []int   `json:"days_of_week,omitempty"`
	PriceModifier float64 `json:"price_modifier"`
	IsActive      bool    `json:"is_active"`
}

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) CreateVenue(ctx context.Context, v *Venue, staffPinHash string) (*Venue, error) {
	out := &Venue{}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO catalog.venues (name, slug, city, address, capacity, staff_pin, timezone, theme)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING venue_id, name, slug, city, COALESCE(address,''), capacity, timezone, theme, is_active, created_at
	`, v.Name, v.Slug, v.City, v.Address, v.Capacity, staffPinHash, v.Timezone, v.Theme,
	).Scan(&out.VenueID, &out.Name, &out.Slug, &out.City, &out.Address, &out.Capacity, &out.Timezone, &out.Theme, &out.IsActive, &out.CreatedAt)
	if err != nil {
		if isUniq(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create venue: %w", err)
	}
	return out, nil
}

func (s *Store) GetVenue(ctx context.Context, venueID string) (*Venue, error) {
	out := &Venue{}
	err := s.db.QueryRowContext(ctx, `
		SELECT venue_id, name, slug, city, COALESCE(address,''), capacity, timezone, theme,
		       COALESCE(stripe_account,''), is_active, created_at
		FROM catalog.venues WHERE venue_id = $1
	`, venueID).Scan(&out.VenueID, &out.Name, &out.Slug, &out.City, &out.Address,
		&out.Capacity, &out.Timezone, &out.Theme, &out.StripeAccount, &out.IsActive, &out.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return out, err
}

func (s *Store) ListVenues(ctx context.Context) ([]*Venue, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT venue_id, name, slug, city, COALESCE(address,''), capacity, timezone, theme, is_active, created_at
		FROM catalog.venues WHERE is_active = true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vs []*Venue
	for rows.Next() {
		v := &Venue{}
		if err := rows.Scan(&v.VenueID, &v.Name, &v.Slug, &v.City, &v.Address,
			&v.Capacity, &v.Timezone, &v.Theme, &v.IsActive, &v.CreatedAt); err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, rows.Err()
}

func (s *Store) ListItems(ctx context.Context, venueID string, activeOnly bool) ([]*CatalogItem, error) {
	q := `SELECT item_id, venue_id, name, category, price_nc, COALESCE(icon,''),
	             stock_qty, low_threshold, is_active, display_order, happy_hour_price_nc, created_at, updated_at
	      FROM catalog.catalog_items WHERE venue_id = $1`
	if activeOnly {
		q += " AND is_active = true"
	}
	q += " ORDER BY display_order, name"
	rows, err := s.db.QueryContext(ctx, q, venueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []*CatalogItem
	for rows.Next() {
		i := &CatalogItem{}
		if err := rows.Scan(&i.ItemID, &i.VenueID, &i.Name, &i.Category, &i.PriceNC,
			&i.Icon, &i.StockQty, &i.LowThreshold, &i.IsActive, &i.DisplayOrder,
			&i.HappyHourPriceNC, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (s *Store) GetItem(ctx context.Context, itemID string) (*CatalogItem, error) {
	i := &CatalogItem{}
	err := s.db.QueryRowContext(ctx, `
		SELECT item_id, venue_id, name, category, price_nc, COALESCE(icon,''),
		       stock_qty, low_threshold, is_active, display_order, happy_hour_price_nc, created_at, updated_at
		FROM catalog.catalog_items WHERE item_id = $1
	`, itemID).Scan(&i.ItemID, &i.VenueID, &i.Name, &i.Category, &i.PriceNC,
		&i.Icon, &i.StockQty, &i.LowThreshold, &i.IsActive, &i.DisplayOrder,
		&i.HappyHourPriceNC, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return i, err
}

func (s *Store) CreateItem(ctx context.Context, i *CatalogItem) (*CatalogItem, error) {
	out := &CatalogItem{}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO catalog.catalog_items (venue_id, name, category, price_nc, icon, stock_qty, display_order, happy_hour_price_nc)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8)
		RETURNING item_id, venue_id, name, category, price_nc, COALESCE(icon,''),
		          stock_qty, low_threshold, is_active, display_order, happy_hour_price_nc, created_at, updated_at
	`, i.VenueID, i.Name, i.Category, i.PriceNC, i.Icon, i.StockQty, i.DisplayOrder, i.HappyHourPriceNC,
	).Scan(&out.ItemID, &out.VenueID, &out.Name, &out.Category, &out.PriceNC,
		&out.Icon, &out.StockQty, &out.LowThreshold, &out.IsActive, &out.DisplayOrder,
		&out.HappyHourPriceNC, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create item: %w", err)
	}
	return out, nil
}

func (s *Store) DeleteItem(ctx context.Context, itemID string) error {
	r, err := s.db.ExecContext(ctx, `UPDATE catalog.catalog_items SET is_active=false WHERE item_id=$1`, itemID)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ListEvents(ctx context.Context, venueID string) ([]*Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT event_id, venue_id, title, starts_at, ends_at, COALESCE(description,''),
		       COALESCE(genre,''), COALESCE(image_url,''), is_public, created_at
		FROM catalog.events WHERE venue_id=$1 ORDER BY starts_at DESC`, venueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var evts []*Event
	for rows.Next() {
		e := &Event{}
		if err := rows.Scan(&e.EventID, &e.VenueID, &e.Title, &e.StartsAt, &e.EndsAt,
			&e.Description, &e.Genre, &e.ImageURL, &e.IsPublic, &e.CreatedAt); err != nil {
			return nil, err
		}
		evts = append(evts, e)
	}
	return evts, rows.Err()
}

func (s *Store) CreateEvent(ctx context.Context, e *Event) (*Event, error) {
	out := &Event{}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO catalog.events (venue_id, title, starts_at, ends_at, description, genre, image_url, is_public)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),$8)
		RETURNING event_id, venue_id, title, starts_at, ends_at,
		          COALESCE(description,''), COALESCE(genre,''), COALESCE(image_url,''), is_public, created_at
	`, e.VenueID, e.Title, e.StartsAt, e.EndsAt, e.Description, e.Genre, e.ImageURL, e.IsPublic,
	).Scan(&out.EventID, &out.VenueID, &out.Title, &out.StartsAt, &out.EndsAt,
		&out.Description, &out.Genre, &out.ImageURL, &out.IsPublic, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}
	return out, nil
}

func (s *Store) ListHappyHours(ctx context.Context, venueID string) ([]*HappyHourRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT rule_id, venue_id, name, starts_at::text, ends_at::text,
		       days_of_week, price_modifier, is_active
		FROM catalog.happy_hour_rules WHERE venue_id=$1 AND is_active=true`, venueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []*HappyHourRule
	for rows.Next() {
		r := &HappyHourRule{}
		var days []byte
		if err := rows.Scan(&r.RuleID, &r.VenueID, &r.Name, &r.StartsAt, &r.EndsAt,
			&days, &r.PriceModifier, &r.IsActive); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, rows.Err()
}

func (s *Store) CreateHappyHour(ctx context.Context, r *HappyHourRule) (*HappyHourRule, error) {
	out := &HappyHourRule{}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO catalog.happy_hour_rules (venue_id, name, starts_at, ends_at, price_modifier)
		VALUES ($1,$2,$3::time,$4::time,$5)
		RETURNING rule_id, venue_id, name, starts_at::text, ends_at::text, price_modifier, is_active
	`, r.VenueID, r.Name, r.StartsAt, r.EndsAt, r.PriceModifier,
	).Scan(&out.RuleID, &out.VenueID, &out.Name, &out.StartsAt, &out.EndsAt, &out.PriceModifier, &out.IsActive)
	if err != nil {
		return nil, fmt.Errorf("create happy hour: %w", err)
	}
	return out, nil
}

func isUniq(err error) bool {
	s := err.Error()
	return contains(s, "23505") || contains(s, "duplicate key") || contains(s, "unique constraint")
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
