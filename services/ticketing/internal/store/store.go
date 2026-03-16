// Package store provides Postgres access for the ticketing service.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("not found")

type Ticket struct {
	TicketID   string     `json:"ticket_id"`
	UserID     string     `json:"user_id"`
	EventID    string     `json:"event_id"`
	VenueID    string     `json:"venue_id"`
	PriceNC    int        `json:"price_nc"`
	Status     string     `json:"status"` // "valid" | "used" | "refunded"
	QRCode     string     `json:"qr_code"`
	IssuedAt   time.Time  `json:"issued_at"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
}

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) IssueTicket(ctx context.Context, userID, eventID, venueID, iKey, qrCode string, priceNC int) (*Ticket, error) {
	t := &Ticket{}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO ticketing.tickets (user_id, event_id, venue_id, price_nc, qr_code, idempotency_key)
		VALUES ($1, $2::uuid, $3::uuid, $4, $5, $6)
		ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key
		RETURNING ticket_id, user_id, event_id::text, venue_id::text, price_nc,
		          status, qr_code, issued_at, used_at, idempotency_key
	`, userID, eventID, venueID, priceNC, qrCode, iKey,
	).Scan(&t.TicketID, &t.UserID, &t.EventID, &t.VenueID, &t.PriceNC,
		&t.Status, &t.QRCode, &t.IssuedAt, &t.UsedAt, &t.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("issue ticket: %w", err)
	}
	return t, nil
}

func (s *Store) GetTicket(ctx context.Context, ticketID string) (*Ticket, error) {
	t := &Ticket{}
	err := s.db.QueryRowContext(ctx, `
		SELECT ticket_id, user_id, event_id::text, venue_id::text, price_nc,
		       status, qr_code, issued_at, used_at, idempotency_key
		FROM ticketing.tickets WHERE ticket_id = $1
	`, ticketID).Scan(&t.TicketID, &t.UserID, &t.EventID, &t.VenueID, &t.PriceNC,
		&t.Status, &t.QRCode, &t.IssuedAt, &t.UsedAt, &t.IdempotencyKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Store) GetTicketByQR(ctx context.Context, qrCode string) (*Ticket, error) {
	t := &Ticket{}
	err := s.db.QueryRowContext(ctx, `
		SELECT ticket_id, user_id, event_id::text, venue_id::text, price_nc,
		       status, qr_code, issued_at, used_at, idempotency_key
		FROM ticketing.tickets WHERE qr_code = $1
	`, qrCode).Scan(&t.TicketID, &t.UserID, &t.EventID, &t.VenueID, &t.PriceNC,
		&t.Status, &t.QRCode, &t.IssuedAt, &t.UsedAt, &t.IdempotencyKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Store) UseTicket(ctx context.Context, ticketID string) (*Ticket, error) {
	now := time.Now()
	t := &Ticket{}
	err := s.db.QueryRowContext(ctx, `
		UPDATE ticketing.tickets SET status='used', used_at=$1
		WHERE ticket_id=$2 AND status='valid'
		RETURNING ticket_id, user_id, event_id::text, venue_id::text, price_nc,
		          status, qr_code, issued_at, used_at, idempotency_key
	`, now, ticketID).Scan(&t.TicketID, &t.UserID, &t.EventID, &t.VenueID, &t.PriceNC,
		&t.Status, &t.QRCode, &t.IssuedAt, &t.UsedAt, &t.IdempotencyKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *Store) ListUserTickets(ctx context.Context, userID string) ([]*Ticket, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ticket_id, user_id, event_id::text, venue_id::text, price_nc,
		       status, qr_code, issued_at, used_at, idempotency_key
		FROM ticketing.tickets WHERE user_id = $1 ORDER BY issued_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tickets []*Ticket
	for rows.Next() {
		t := &Ticket{}
		if err := rows.Scan(&t.TicketID, &t.UserID, &t.EventID, &t.VenueID, &t.PriceNC,
			&t.Status, &t.QRCode, &t.IssuedAt, &t.UsedAt, &t.IdempotencyKey); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}
