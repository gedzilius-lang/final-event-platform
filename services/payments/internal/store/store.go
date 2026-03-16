// Package store provides Postgres access for the payments service.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"niteos.internal/payments/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

// CreateTopup inserts a pending topup record. Idempotent on idempotency_key.
func (s *Store) CreateTopup(ctx context.Context, userID, provider, intentID, iKey string, amountCHF float64) (*domain.Topup, error) {
	t := &domain.Topup{}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO payments.topups
		    (user_id, amount_chf, amount_nc, provider, provider_intent_id, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (idempotency_key) DO UPDATE
		    SET idempotency_key = EXCLUDED.idempotency_key
		RETURNING topup_id, user_id, amount_chf, amount_nc, provider,
		          provider_intent_id, status, idempotency_key, created_at, confirmed_at
	`, userID, amountCHF, domain.CHFToNC(amountCHF), provider, intentID, iKey,
	).Scan(&t.TopupID, &t.UserID, &t.AmountCHF, &t.AmountNC, &t.Provider,
		&t.ProviderIntentID, &t.Status, &t.IdempotencyKey, &t.CreatedAt, &t.ConfirmedAt)
	if err != nil {
		return nil, fmt.Errorf("create topup: %w", err)
	}
	return t, nil
}

// ConfirmTopup transitions a pending topup to confirmed.
// Returns ErrNotFound if already confirmed (idempotent for webhook replay).
func (s *Store) ConfirmTopup(ctx context.Context, intentID string) (*domain.Topup, error) {
	now := time.Now()
	t := &domain.Topup{}
	err := s.db.QueryRowContext(ctx, `
		UPDATE payments.topups SET status='confirmed', confirmed_at=$1
		WHERE provider_intent_id=$2 AND status='pending'
		RETURNING topup_id, user_id, amount_chf, amount_nc, provider,
		          provider_intent_id, status, idempotency_key, created_at, confirmed_at
	`, now, intentID).Scan(&t.TopupID, &t.UserID, &t.AmountCHF, &t.AmountNC, &t.Provider,
		&t.ProviderIntentID, &t.Status, &t.IdempotencyKey, &t.CreatedAt, &t.ConfirmedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// Either not found or already confirmed — fetch current record
		return s.GetTopupByIntentID(ctx, intentID)
	}
	if err != nil {
		return nil, fmt.Errorf("confirm topup: %w", err)
	}
	return t, nil
}

// GetTopup fetches a topup by ID.
func (s *Store) GetTopup(ctx context.Context, topupID string) (*domain.Topup, error) {
	t := &domain.Topup{}
	err := s.db.QueryRowContext(ctx, `
		SELECT topup_id, user_id, amount_chf, amount_nc, provider,
		       provider_intent_id, status, idempotency_key, created_at, confirmed_at
		FROM payments.topups WHERE topup_id = $1
	`, topupID).Scan(&t.TopupID, &t.UserID, &t.AmountCHF, &t.AmountNC, &t.Provider,
		&t.ProviderIntentID, &t.Status, &t.IdempotencyKey, &t.CreatedAt, &t.ConfirmedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// GetTopupByIntentID fetches a topup by provider intent ID.
func (s *Store) GetTopupByIntentID(ctx context.Context, intentID string) (*domain.Topup, error) {
	t := &domain.Topup{}
	err := s.db.QueryRowContext(ctx, `
		SELECT topup_id, user_id, amount_chf, amount_nc, provider,
		       provider_intent_id, status, idempotency_key, created_at, confirmed_at
		FROM payments.topups WHERE provider_intent_id = $1
	`, intentID).Scan(&t.TopupID, &t.UserID, &t.AmountCHF, &t.AmountNC, &t.Provider,
		&t.ProviderIntentID, &t.Status, &t.IdempotencyKey, &t.CreatedAt, &t.ConfirmedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}
