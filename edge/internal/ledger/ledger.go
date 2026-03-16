// Package ledger provides the local append-only ledger for edge devices.
// Same invariants as cloud ledger: INSERT only, idempotent writes, balance projection.
package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrZeroAmount = errors.New("amount_nc must not be zero")
var ErrMissingKey = errors.New("idempotency_key required")
var ErrNotFound = errors.New("not found")

// Event mirrors cloud ledger_events for local storage.
type Event struct {
	EventID        string
	EventType      string
	UserID         string
	VenueID        string
	DeviceID       string
	AmountNC       int
	ReferenceID    string
	IdempotencyKey string
	OccurredAt     time.Time
	SyncStatus     string
}

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

// WriteEvent writes a ledger event locally and enqueues it for sync.
// Returns existing event if idempotency_key already exists.
func (s *Store) WriteEvent(ctx context.Context, e *Event) (*Event, error) {
	if e.AmountNC == 0 {
		return nil, ErrZeroAmount
	}
	if e.IdempotencyKey == "" {
		return nil, ErrMissingKey
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	writtenBy := e.EventType // use event_type as proxy for author service
	out := &Event{}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO ledger_events
		    (event_type, user_id, venue_id, device_id, amount_nc,
		     reference_id, idempotency_key, written_by)
		VALUES (?, ?, NULLIF(?,''), NULLIF(?,''), ?, NULLIF(?,''), ?, ?)
		ON CONFLICT(idempotency_key) DO UPDATE
		    SET idempotency_key=EXCLUDED.idempotency_key
		RETURNING event_id, event_type, user_id, COALESCE(venue_id,''),
		          COALESCE(device_id,''), amount_nc,
		          COALESCE(reference_id,''), idempotency_key, occurred_at, sync_status
	`, e.EventType, e.UserID, e.VenueID, e.DeviceID, e.AmountNC, e.ReferenceID, e.IdempotencyKey, writtenBy,
	).Scan(&out.EventID, &out.EventType, &out.UserID, &out.VenueID, &out.DeviceID,
		&out.AmountNC, &out.ReferenceID, &out.IdempotencyKey, &out.OccurredAt, &out.SyncStatus)
	if err != nil {
		return nil, fmt.Errorf("write event: %w", err)
	}

	// Enqueue for sync
	_, _ = tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO sync_queue (idempotency_key) VALUES (?)`,
		e.IdempotencyKey)

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetBalance returns the spendable balance for a user at this venue.
// Excludes: topup_pending, venue_checkin, session_closed.
func (s *Store) GetBalance(ctx context.Context, userID string) (int, error) {
	var bal int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount_nc), 0)
		FROM ledger_events
		WHERE user_id = ?
		  AND event_type NOT IN ('topup_pending','venue_checkin','session_closed')
	`, userID).Scan(&bal)
	return bal, err
}

// PendingSyncEvents returns up to limit events pending sync, oldest first.
func (s *Store) PendingSyncEvents(ctx context.Context, limit int) ([]*Event, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.event_id, l.event_type, l.user_id, COALESCE(l.venue_id,''),
		       COALESCE(l.device_id,''), l.amount_nc,
		       COALESCE(l.reference_id,''), l.idempotency_key, l.occurred_at, l.sync_status
		FROM ledger_events l
		JOIN sync_queue q ON l.idempotency_key = q.idempotency_key
		WHERE q.status = 'pending'
		ORDER BY l.occurred_at ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []*Event
	for rows.Next() {
		e := &Event{}
		if err := rows.Scan(&e.EventID, &e.EventType, &e.UserID, &e.VenueID, &e.DeviceID,
			&e.AmountNC, &e.ReferenceID, &e.IdempotencyKey, &e.OccurredAt, &e.SyncStatus); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// MarkSynced marks a batch of events as synced.
func (s *Store) MarkSynced(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	now := time.Now()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, k := range keys {
		_, err = tx.ExecContext(ctx,
			`UPDATE sync_queue SET status='synced', last_attempt_at=? WHERE idempotency_key=?`, now, k)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE ledger_events SET sync_status='synced', synced_at=? WHERE idempotency_key=?`, now, k)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

// IncrementSyncAttempt records a failed sync attempt.
func (s *Store) IncrementSyncAttempt(ctx context.Context, key string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`UPDATE sync_queue SET attempts=attempts+1, last_attempt_at=? WHERE idempotency_key=?`,
		now, key)
	return err
}
