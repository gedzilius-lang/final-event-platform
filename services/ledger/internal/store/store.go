// Package store provides Postgres access for the ledger service.
// INSERT only — UPDATE and DELETE are revoked at the DB level for niteos_app.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"niteos.internal/ledger/internal/domain"
)

// Store wraps the Postgres connection for the ledger service.
type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store { return &Store{db: db} }

// WriteEvent inserts a new ledger event.
// Returns the existing event (idempotent) if the idempotency_key already exists.
func (s *Store) WriteEvent(ctx context.Context, req *domain.WriteRequest, writtenBy string) (*domain.LedgerEvent, error) {
	e := &domain.LedgerEvent{}
	var venueID, deviceID, referenceID sql.NullString
	var amountCHF sql.NullFloat64

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO ledger.ledger_events
		    (event_type, user_id, venue_id, device_id, amount_nc, amount_chf,
		     reference_id, idempotency_key, synced_from, written_by)
		VALUES ($1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5,$6,
		        NULLIF($7,'')::uuid,$8,COALESCE(NULLIF($9,''),'cloud'),$10)
		ON CONFLICT (idempotency_key) DO UPDATE
		    SET idempotency_key = EXCLUDED.idempotency_key  -- no-op, forces RETURNING
		RETURNING event_id, event_type, user_id,
		          COALESCE(venue_id::text,''), COALESCE(device_id::text,''),
		          amount_nc, amount_chf, COALESCE(reference_id::text,''),
		          idempotency_key, occurred_at, synced_from, written_by
	`,
		req.EventType, req.UserID, req.VenueID, req.DeviceID,
		req.AmountNC, req.AmountCHF, req.ReferenceID,
		req.IdempotencyKey, req.SyncedFrom, writtenBy,
	).Scan(
		&e.EventID, &e.EventType, &e.UserID,
		&venueID, &deviceID,
		&e.AmountNC, &amountCHF, &referenceID,
		&e.IdempotencyKey, &e.OccurredAt, &e.SyncedFrom, &e.WrittenBy,
	)
	if err != nil {
		return nil, fmt.Errorf("write event: %w", err)
	}
	e.VenueID = venueID.String
	e.DeviceID = deviceID.String
	e.ReferenceID = referenceID.String
	if amountCHF.Valid {
		v := amountCHF.Float64
		e.AmountCHF = &v
	}
	return e, nil
}

// GetBalance returns the spendable balance for a user (global or venue-scoped).
// Balance excludes: topup_pending, venue_checkin, session_closed.
func (s *Store) GetBalance(ctx context.Context, userID, venueID string) (int, error) {
	var query string
	var args []any

	if venueID == "" {
		query = `
			SELECT COALESCE(SUM(amount_nc), 0)
			FROM ledger.ledger_events
			WHERE user_id = $1
			  AND event_type NOT IN ('topup_pending','venue_checkin','session_closed')`
		args = []any{userID}
	} else {
		query = `
			SELECT COALESCE(SUM(amount_nc), 0)
			FROM ledger.ledger_events
			WHERE user_id = $1
			  AND event_type NOT IN ('topup_pending','venue_checkin','session_closed')
			  AND (venue_id = $2::uuid OR venue_id IS NULL)`
		args = []any{userID, venueID}
	}

	var balance int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&balance); err != nil {
		return 0, fmt.Errorf("get balance: %w", err)
	}
	return balance, nil
}

// ListUserEvents returns paginated ledger events for a user, newest first.
func (s *Store) ListUserEvents(ctx context.Context, userID string, limit, offset int) ([]*domain.LedgerEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT event_id, event_type, user_id,
		       COALESCE(venue_id::text,''), COALESCE(device_id::text,''),
		       amount_nc, amount_chf, COALESCE(reference_id::text,''),
		       idempotency_key, occurred_at, synced_from, written_by
		FROM ledger.ledger_events
		WHERE user_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list user events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// ListVenueEvents returns paginated ledger events for a venue, newest first.
func (s *Store) ListVenueEvents(ctx context.Context, venueID string, limit, offset int) ([]*domain.LedgerEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT event_id, event_type, user_id,
		       COALESCE(venue_id::text,''), COALESCE(device_id::text,''),
		       amount_nc, amount_chf, COALESCE(reference_id::text,''),
		       idempotency_key, occurred_at, synced_from, written_by
		FROM ledger.ledger_events
		WHERE venue_id = $1::uuid
		ORDER BY occurred_at DESC
		LIMIT $2 OFFSET $3
	`, venueID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list venue events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// GetEventByIdempotencyKey fetches an event by idempotency key (for idempotency checks by callers).
func (s *Store) GetEventByIdempotencyKey(ctx context.Context, key string) (*domain.LedgerEvent, error) {
	e := &domain.LedgerEvent{}
	var venueID, deviceID, referenceID sql.NullString
	var amountCHF sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT event_id, event_type, user_id,
		       COALESCE(venue_id::text,''), COALESCE(device_id::text,''),
		       amount_nc, amount_chf, COALESCE(reference_id::text,''),
		       idempotency_key, occurred_at, synced_from, written_by
		FROM ledger.ledger_events WHERE idempotency_key = $1
	`, key).Scan(
		&e.EventID, &e.EventType, &e.UserID,
		&venueID, &deviceID,
		&e.AmountNC, &amountCHF, &referenceID,
		&e.IdempotencyKey, &e.OccurredAt, &e.SyncedFrom, &e.WrittenBy,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	e.VenueID = venueID.String
	e.DeviceID = deviceID.String
	e.ReferenceID = referenceID.String
	if amountCHF.Valid {
		v := amountCHF.Float64
		e.AmountCHF = &v
	}
	return e, nil
}

func scanEvents(rows *sql.Rows) ([]*domain.LedgerEvent, error) {
	var events []*domain.LedgerEvent
	for rows.Next() {
		e := &domain.LedgerEvent{}
		var venueID, deviceID, referenceID sql.NullString
		var amountCHF sql.NullFloat64
		if err := rows.Scan(
			&e.EventID, &e.EventType, &e.UserID,
			&venueID, &deviceID,
			&e.AmountNC, &amountCHF, &referenceID,
			&e.IdempotencyKey, &e.OccurredAt, &e.SyncedFrom, &e.WrittenBy,
		); err != nil {
			return nil, err
		}
		e.VenueID = venueID.String
		e.DeviceID = deviceID.String
		e.ReferenceID = referenceID.String
		if amountCHF.Valid {
			v := amountCHF.Float64
			e.AmountCHF = &v
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
