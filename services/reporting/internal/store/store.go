// Package store provides read-only Postgres queries for the reporting service.
// All queries are read-only. No writes occur here.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

// VenueRevenue is aggregated revenue for a venue in a time window.
// Field names match the admin console TypeScript interface.
type VenueRevenue struct {
	VenueID       string `json:"venue_id"`
	PeriodStart   string `json:"period_start"`
	PeriodEnd     string `json:"period_end"`
	TotalOrdersNC int    `json:"total_orders_nc"`
	TotalTopupsNC int    `json:"total_topups_nc"`
	OrderCount    int    `json:"order_count"`
	SessionCount  int    `json:"session_count"`
}

// GetVenueRevenue aggregates ledger events and session data for a venue in a date range.
func (s *Store) GetVenueRevenue(ctx context.Context, venueID string, from, to time.Time) (*VenueRevenue, error) {
	r := &VenueRevenue{
		VenueID:     venueID,
		PeriodStart: from.Format("2006-01-02"),
		PeriodEnd:   to.Format("2006-01-02"),
	}

	// Aggregate order spend and topup credits from ledger events.
	// order_paid events have negative amount_nc (debit), so negate for revenue.
	err := s.db.QueryRowContext(ctx, `
		SELECT
		    COALESCE(SUM(CASE WHEN event_type='order_paid' THEN -amount_nc ELSE 0 END), 0),
		    COALESCE(SUM(CASE WHEN event_type='topup_confirmed' THEN amount_nc ELSE 0 END), 0),
		    COUNT(CASE WHEN event_type='order_paid' THEN 1 END)
		FROM ledger.ledger_events
		WHERE venue_id = $1::uuid
		  AND occurred_at >= $2
		  AND occurred_at < $3
	`, venueID, from, to).Scan(&r.TotalOrdersNC, &r.TotalTopupsNC, &r.OrderCount)
	if err != nil {
		return nil, fmt.Errorf("venue revenue ledger: %w", err)
	}

	// Session count from sessions table.
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM sessions.venue_sessions
		WHERE venue_id = $1
		  AND opened_at >= $2
		  AND opened_at < $3
	`, venueID, from, to).Scan(&r.SessionCount)
	if err != nil {
		return nil, fmt.Errorf("venue revenue sessions: %w", err)
	}

	return r, nil
}

// SessionStats is aggregated session data for a venue.
type SessionStats struct {
	VenueID      string  `json:"venue_id"`
	PeriodFrom   time.Time `json:"period_from"`
	PeriodTo     time.Time `json:"period_to"`
	TotalSessions int     `json:"total_sessions"`
	AvgSpendNC   float64 `json:"avg_spend_nc"`
	TotalSpendNC int     `json:"total_spend_nc"`
}

// GetVenueSessionStats aggregates session data for a venue.
func (s *Store) GetVenueSessionStats(ctx context.Context, venueID string, from, to time.Time) (*SessionStats, error) {
	r := &SessionStats{VenueID: venueID, PeriodFrom: from, PeriodTo: to}

	err := s.db.QueryRowContext(ctx, `
		SELECT
		    COUNT(*) AS total_sessions,
		    COALESCE(AVG(total_spend_nc), 0) AS avg_spend_nc,
		    COALESCE(SUM(total_spend_nc), 0) AS total_spend_nc
		FROM sessions.venue_sessions
		WHERE venue_id = $1
		  AND opened_at >= $2
		  AND opened_at < $3
		  AND status = 'closed'
	`, venueID, from, to).Scan(&r.TotalSessions, &r.AvgSpendNC, &r.TotalSpendNC)
	if err != nil {
		return nil, fmt.Errorf("session stats: %w", err)
	}
	return r, nil
}

// TopupSummary is aggregated topup data.
type TopupSummary struct {
	PeriodFrom   time.Time `json:"period_from"`
	PeriodTo     time.Time `json:"period_to"`
	TotalTopups  int       `json:"total_topups"`
	TotalCHF     float64   `json:"total_chf"`
	TotalNC      int       `json:"total_nc"`
}

// GetTopupSummary aggregates confirmed topups in a time window.
func (s *Store) GetTopupSummary(ctx context.Context, from, to time.Time) (*TopupSummary, error) {
	r := &TopupSummary{PeriodFrom: from, PeriodTo: to}

	// amount_nc is derived: 1 CHF = 1 NC (fixed peg, PRODUCT_BLUEPRINT §NiteCoin)
	err := s.db.QueryRowContext(ctx, `
		SELECT
		    COUNT(*),
		    COALESCE(SUM(amount_chf), 0),
		    COALESCE(SUM(amount_chf)::int, 0)
		FROM payments.payment_intents
		WHERE status IN ('confirmed', 'captured')
		  AND confirmed_at >= $1
		  AND confirmed_at < $2
	`, from, to).Scan(&r.TotalTopups, &r.TotalCHF, &r.TotalNC)
	if err != nil {
		return nil, fmt.Errorf("topup summary: %w", err)
	}
	return r, nil
}
