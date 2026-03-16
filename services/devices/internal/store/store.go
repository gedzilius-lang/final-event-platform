// Package store provides Postgres access for the devices service.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("device already enrolled")

type Device struct {
	DeviceID   string     `json:"device_id"`
	VenueID    string     `json:"venue_id"`
	DeviceRole string     `json:"device_role"` // "terminal" | "tablet" | "kiosk"
	Status     string     `json:"status"`      // "active" | "inactive" | "revoked"
	Name       string     `json:"name"`
	EnrolledAt time.Time  `json:"enrolled_at"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
}

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Enroll(ctx context.Context, venueID, deviceRole, name string) (*Device, error) {
	d := &Device{}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO devices.devices (venue_id, device_role, name)
		VALUES ($1, $2, $3)
		RETURNING device_id, venue_id, device_role, status, name, enrolled_at, last_seen_at
	`, venueID, deviceRole, name).Scan(
		&d.DeviceID, &d.VenueID, &d.DeviceRole, &d.Status, &d.Name, &d.EnrolledAt, &d.LastSeenAt,
	)
	if err != nil {
		return nil, fmt.Errorf("enroll device: %w", err)
	}
	return d, nil
}

func (s *Store) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	d := &Device{}
	err := s.db.QueryRowContext(ctx, `
		SELECT device_id, venue_id, device_role, status, name, enrolled_at, last_seen_at
		FROM devices.devices WHERE device_id = $1
	`, deviceID).Scan(
		&d.DeviceID, &d.VenueID, &d.DeviceRole, &d.Status, &d.Name, &d.EnrolledAt, &d.LastSeenAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

func (s *Store) ListVenueDevices(ctx context.Context, venueID string) ([]*Device, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_id, venue_id, device_role, status, name, enrolled_at, last_seen_at
		FROM devices.devices WHERE venue_id = $1 ORDER BY enrolled_at DESC
	`, venueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var devices []*Device
	for rows.Next() {
		d := &Device{}
		if err := rows.Scan(&d.DeviceID, &d.VenueID, &d.DeviceRole, &d.Status, &d.Name,
			&d.EnrolledAt, &d.LastSeenAt); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (s *Store) Heartbeat(ctx context.Context, deviceID string) error {
	now := time.Now()
	r, err := s.db.ExecContext(ctx,
		`UPDATE devices.devices SET last_seen_at=$1 WHERE device_id=$2 AND status='active'`,
		now, deviceID)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetStatus(ctx context.Context, deviceID, status string) (*Device, error) {
	d := &Device{}
	err := s.db.QueryRowContext(ctx, `
		UPDATE devices.devices SET status=$1 WHERE device_id=$2
		RETURNING device_id, venue_id, device_role, status, name, enrolled_at, last_seen_at
	`, status, deviceID).Scan(
		&d.DeviceID, &d.VenueID, &d.DeviceRole, &d.Status, &d.Name, &d.EnrolledAt, &d.LastSeenAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}
