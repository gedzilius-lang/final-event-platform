// Package sync provides the background sync agent for the edge service.
// The agent reads pending ledger events and POSTs sync frames to the cloud sync service.
package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"niteos.internal/edge/internal/ledger"
)

const (
	batchSize   = 50
	maxAttempts = 10
)

type Agent struct {
	store    *ledger.Store
	syncURL  string
	deviceID string
	venueID  string
}

func NewAgent(store *ledger.Store, syncURL, deviceID, venueID string) *Agent {
	return &Agent{store: store, syncURL: syncURL, deviceID: deviceID, venueID: venueID}
}

// Run starts the sync loop. Blocks until ctx is cancelled.
func (a *Agent) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := a.sync(ctx); err != nil {
				slog.Warn("sync failed", "err", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) sync(ctx context.Context) error {
	events, err := a.store.PendingSyncEvents(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("fetch pending events: %w", err)
	}
	if len(events) == 0 {
		return nil
	}

	frameEvents := make([]map[string]any, 0, len(events))
	for _, e := range events {
		frameEvents = append(frameEvents, map[string]any{
			"event_type":      e.EventType,
			"user_id":         e.UserID,
			"venue_id":        e.VenueID,
			"device_id":       e.DeviceID,
			"amount_nc":       e.AmountNC,
			"reference_id":    e.ReferenceID,
			"idempotency_key": e.IdempotencyKey,
			"occurred_at":     e.OccurredAt,
		})
	}

	frame := map[string]any{
		"device_id":  a.deviceID,
		"venue_id":   a.venueID,
		"frame_id":   fmt.Sprintf("frame_%d", time.Now().UnixNano()),
		"events":     frameEvents,
		"created_at": time.Now(),
	}

	data, _ := json.Marshal(frame)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.syncURL+"/frames", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "edge")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Mark all as failed attempt
		for _, e := range events {
			_ = a.store.IncrementSyncAttempt(ctx, e.IdempotencyKey)
		}
		return fmt.Errorf("sync service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		for _, e := range events {
			_ = a.store.IncrementSyncAttempt(ctx, e.IdempotencyKey)
		}
		return fmt.Errorf("sync service returned %d", resp.StatusCode)
	}

	// Parse results and mark each event accordingly
	var result struct {
		Results []struct {
			IdempotencyKey string `json:"idempotency_key"`
			Status         string `json:"status"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode sync result: %w", err)
	}

	var accepted []string
	for _, r := range result.Results {
		if r.Status == "accepted" || r.Status == "duplicate" {
			accepted = append(accepted, r.IdempotencyKey)
		} else {
			_ = a.store.IncrementSyncAttempt(ctx, r.IdempotencyKey)
		}
	}

	if len(accepted) > 0 {
		if err := a.store.MarkSynced(ctx, accepted); err != nil {
			slog.Error("mark synced", "err", err)
		} else {
			slog.Info("sync complete", "accepted", len(accepted))
		}
	}

	return nil
}
