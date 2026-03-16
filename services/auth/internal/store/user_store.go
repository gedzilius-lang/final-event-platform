package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("not found")

// UserRecord holds the data the auth service needs about a user.
type UserRecord struct {
	UserID       string
	Email        string
	PasswordHash string
	Role         string
}

// UserStore provides read access to the profiles schema.
// Auth service does NOT own the users table; it reads via profiles service API.
// This store is used only for PIN login venue lookups.
type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

// GetVenueStaffPin returns the hashed staff_pin for a venue (for PIN login validation).
func (s *UserStore) GetVenueStaffPin(ctx context.Context, venueID string) (string, error) {
	var pin string
	err := s.db.QueryRowContext(ctx,
		`SELECT staff_pin FROM catalog.venues WHERE venue_id = $1 AND is_active = true`,
		venueID,
	).Scan(&pin)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get venue staff pin: %w", err)
	}
	return pin, nil
}

// GetDeviceVenueRole returns the venue_id and device_role for an enrolled device.
func (s *UserStore) GetDeviceVenueRole(ctx context.Context, deviceID string) (venueID, deviceRole string, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT venue_id::text, device_role FROM devices.devices WHERE device_id = $1 AND status = 'active'`,
		deviceID,
	).Scan(&venueID, &deviceRole)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("get device venue role: %w", err)
	}
	return venueID, deviceRole, nil
}
