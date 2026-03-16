package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("not found")

type VenueSession struct {
	SessionID     string     `json:"session_id"`
	UserID        string     `json:"user_id"`
	VenueID       string     `json:"venue_id"`
	NiteTapUID    string     `json:"nitetap_uid,omitempty"`
	TicketUsed    string     `json:"ticket_used,omitempty"`
	OpenedAt      time.Time  `json:"opened_at"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	TotalSpendNC  int        `json:"total_spend_nc"`
	CheckinDevice string     `json:"checkin_device,omitempty"`
	Status        string     `json:"status"`
}

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) OpenSession(ctx context.Context, userID, venueID, nfcUID, deviceID string) (*VenueSession, error) {
	out := &VenueSession{}
	var tapUID, devID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO sessions.venue_sessions (user_id, venue_id, nitetap_uid, checkin_device)
		VALUES ($1,$2,NULLIF($3,''),NULLIF($4,'')::uuid)
		RETURNING session_id, user_id, venue_id, COALESCE(nitetap_uid,''),
		          COALESCE(ticket_used::text,''), opened_at, closed_at, total_spend_nc,
		          COALESCE(checkin_device::text,''), status
	`, userID, venueID, nfcUID, deviceID).Scan(
		&out.SessionID, &out.UserID, &out.VenueID, &tapUID, &out.TicketUsed,
		&out.OpenedAt, &out.ClosedAt, &out.TotalSpendNC, &devID, &out.Status,
	)
	if err != nil { return nil, fmt.Errorf("open session: %w", err) }
	out.NiteTapUID = tapUID.String
	out.CheckinDevice = devID.String
	return out, nil
}

func (s *Store) CloseSession(ctx context.Context, sessionID string) (*VenueSession, error) {
	now := time.Now()
	out := &VenueSession{}
	err := s.db.QueryRowContext(ctx, `
		UPDATE sessions.venue_sessions SET status='closed', closed_at=$1
		WHERE session_id=$2 AND status='open'
		RETURNING session_id, user_id, venue_id, COALESCE(nitetap_uid,''),
		          opened_at, closed_at, total_spend_nc, COALESCE(checkin_device::text,''), status
	`, now, sessionID).Scan(
		&out.SessionID, &out.UserID, &out.VenueID, &out.NiteTapUID,
		&out.OpenedAt, &out.ClosedAt, &out.TotalSpendNC, &out.CheckinDevice, &out.Status,
	)
	if errors.Is(err, sql.ErrNoRows) { return nil, ErrNotFound }
	if err != nil { return nil, fmt.Errorf("close session: %w", err) }
	return out, nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (*VenueSession, error) {
	out := &VenueSession{}
	err := s.db.QueryRowContext(ctx, `
		SELECT session_id, user_id, venue_id, COALESCE(nitetap_uid,''),
		       opened_at, closed_at, total_spend_nc, COALESCE(checkin_device::text,''), status
		FROM sessions.venue_sessions WHERE session_id=$1
	`, sessionID).Scan(
		&out.SessionID, &out.UserID, &out.VenueID, &out.NiteTapUID,
		&out.OpenedAt, &out.ClosedAt, &out.TotalSpendNC, &out.CheckinDevice, &out.Status,
	)
	if errors.Is(err, sql.ErrNoRows) { return nil, ErrNotFound }
	if err != nil { return nil, fmt.Errorf("get session: %w", err) }
	return out, nil
}

func (s *Store) ListActiveSessions(ctx context.Context, venueID string) ([]*VenueSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT session_id, user_id, venue_id, COALESCE(nitetap_uid,''),
		       opened_at, closed_at, total_spend_nc, COALESCE(checkin_device::text,''), status
		FROM sessions.venue_sessions WHERE venue_id=$1 AND status='open'
		ORDER BY opened_at DESC`, venueID)
	if err != nil { return nil, err }
	defer rows.Close()
	var ss []*VenueSession
	for rows.Next() {
		out := &VenueSession{}
		if err := rows.Scan(&out.SessionID, &out.UserID, &out.VenueID, &out.NiteTapUID,
			&out.OpenedAt, &out.ClosedAt, &out.TotalSpendNC, &out.CheckinDevice, &out.Status); err != nil {
			return nil, err
		}
		ss = append(ss, out)
	}
	return ss, rows.Err()
}

func (s *Store) IncrementSpend(ctx context.Context, sessionID string, amountNC int) error {
	r, err := s.db.ExecContext(ctx,
		`UPDATE sessions.venue_sessions SET total_spend_nc = total_spend_nc + $1 WHERE session_id=$2`,
		amountNC, sessionID)
	if err != nil { return err }
	n, _ := r.RowsAffected()
	if n == 0 { return ErrNotFound }
	return nil
}
