// Package store provides database access for the profiles service.
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

// User represents a NiteOS user record.
type User struct {
	UserID       string     `json:"user_id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"password_hash,omitempty"` // omitted in public responses
	DisplayName  string     `json:"display_name"`
	Role         string     `json:"role"`
	VenueID      string     `json:"venue_id,omitempty"` // set for venue-scoped roles
	GlobalXP     int        `json:"global_xp"`
	GlobalLevel  int        `json:"global_level"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// VenueProfile represents a user's per-venue profile.
type VenueProfile struct {
	ProfileID    string     `json:"profile_id"`
	UserID       string     `json:"user_id"`
	VenueID      string     `json:"venue_id"`
	LocalXP      int        `json:"local_xp"`
	LocalLevel   int        `json:"local_level"`
	FirstVisitAt *time.Time `json:"first_visit_at,omitempty"`
	LastVisitAt  *time.Time `json:"last_visit_at,omitempty"`
	VisitCount   int        `json:"visit_count"`
}

// NiteTap represents an NFC tap card record.
type NiteTap struct {
	TapID        string     `json:"tap_id"`
	NFCUID       string     `json:"nfc_uid"`
	UserID       *string    `json:"user_id,omitempty"`
	VenueID      *string    `json:"venue_id,omitempty"`
	IsAnonymous  bool       `json:"is_anonymous"`
	Status       string     `json:"status"`
	IssuedAt     time.Time  `json:"issued_at"`
	RegisteredAt *time.Time `json:"registered_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

// Store provides Postgres access for the profiles service.
type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// ── Users ─────────────────────────────────────────────────────────────────────

// SetUserVenue updates venue_id and role for a user (nitecore admin operation).
// Pass venueID="" to clear the venue association (sets NULL).
// Pass role="" to leave the role unchanged.
func (s *Store) SetUserVenue(ctx context.Context, userID, venueID, role string) error {
	var venueVal interface{}
	if venueID != "" {
		venueVal = venueID
	}
	var result sql.Result
	var err error
	if role != "" {
		result, err = s.db.ExecContext(ctx, `
			UPDATE profiles.users SET venue_id = $2, role = $3, updated_at = now()
			WHERE user_id = $1
		`, userID, venueVal, role)
	} else {
		result, err = s.db.ExecContext(ctx, `
			UPDATE profiles.users SET venue_id = $2, updated_at = now()
			WHERE user_id = $1
		`, userID, venueVal)
	}
	if err != nil {
		return fmt.Errorf("set user venue: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateUser inserts a new user. Returns ErrConflict on duplicate email.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName string) (*User, error) {
	user := &User{}
	var venueID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO profiles.users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING user_id, email, display_name, role, venue_id, global_xp, global_level, created_at, updated_at
	`, email, passwordHash, displayName).Scan(
		&user.UserID, &user.Email, &user.DisplayName, &user.Role,
		&venueID, &user.GlobalXP, &user.GlobalLevel, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	user.VenueID = venueID.String
	return user, nil
}

// GetUser fetches a user by ID. Password hash is NOT included.
func (s *Store) GetUser(ctx context.Context, userID string) (*User, error) {
	user := &User{}
	var venueID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, email, display_name, role, venue_id, global_xp, global_level, created_at, updated_at
		FROM profiles.users WHERE user_id = $1
	`, userID).Scan(
		&user.UserID, &user.Email, &user.DisplayName, &user.Role,
		&venueID, &user.GlobalXP, &user.GlobalLevel, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	user.VenueID = venueID.String
	return user, nil
}

// ListUsers fetches users, optionally filtered by email prefix. Nitecore only.
func (s *Store) ListUsers(ctx context.Context, emailQuery string) ([]*User, error) {
	var rows *sql.Rows
	var err error
	if emailQuery != "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT user_id, email, display_name, role, venue_id, global_xp, global_level, created_at, updated_at
			FROM profiles.users WHERE email ILIKE $1 ORDER BY created_at DESC LIMIT 100
		`, "%"+emailQuery+"%")
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT user_id, email, display_name, role, venue_id, global_xp, global_level, created_at, updated_at
			FROM profiles.users ORDER BY created_at DESC LIMIT 100
		`)
	}
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := &User{}
		var venueID sql.NullString
		if err := rows.Scan(&u.UserID, &u.Email, &u.DisplayName, &u.Role,
			&venueID, &u.GlobalXP, &u.GlobalLevel, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.VenueID = venueID.String
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetUserByEmail fetches a user by email INCLUDING password hash (internal only).
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	var venueID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, email, password_hash, display_name, role, venue_id, global_xp, global_level, created_at, updated_at
		FROM profiles.users WHERE email = $1
	`, email).Scan(
		&user.UserID, &user.Email, &user.PasswordHash, &user.DisplayName, &user.Role,
		&venueID, &user.GlobalXP, &user.GlobalLevel, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	user.VenueID = venueID.String
	return user, nil
}

// ── VenueProfiles ──────────────────────────────────────────────────────────────

// GetVenueProfile fetches the per-venue profile for a user.
func (s *Store) GetVenueProfile(ctx context.Context, userID, venueID string) (*VenueProfile, error) {
	vp := &VenueProfile{}
	err := s.db.QueryRowContext(ctx, `
		SELECT profile_id, user_id, venue_id, local_xp, local_level,
		       first_visit_at, last_visit_at, visit_count
		FROM profiles.venue_profiles
		WHERE user_id = $1 AND venue_id = $2
	`, userID, venueID).Scan(
		&vp.ProfileID, &vp.UserID, &vp.VenueID, &vp.LocalXP, &vp.LocalLevel,
		&vp.FirstVisitAt, &vp.LastVisitAt, &vp.VisitCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get venue profile: %w", err)
	}
	return vp, nil
}

// UpsertVenueProfile creates or updates the per-venue profile.
// Called by sessions service on check-in.
type UpsertVenueProfileParams struct {
	UserID  string
	VenueID string
}

func (s *Store) UpsertVenueProfile(ctx context.Context, p UpsertVenueProfileParams) (*VenueProfile, error) {
	now := time.Now()
	vp := &VenueProfile{}
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO profiles.venue_profiles (user_id, venue_id, first_visit_at, last_visit_at, visit_count)
		VALUES ($1, $2, $3, $3, 1)
		ON CONFLICT (user_id, venue_id) DO UPDATE
		SET last_visit_at = $3,
		    visit_count   = profiles.venue_profiles.visit_count + 1
		RETURNING profile_id, user_id, venue_id, local_xp, local_level,
		          first_visit_at, last_visit_at, visit_count
	`, p.UserID, p.VenueID, now).Scan(
		&vp.ProfileID, &vp.UserID, &vp.VenueID, &vp.LocalXP, &vp.LocalLevel,
		&vp.FirstVisitAt, &vp.LastVisitAt, &vp.VisitCount,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert venue profile: %w", err)
	}
	return vp, nil
}

// ── XP / Level ────────────────────────────────────────────────────────────────

// XPPerLevel is the number of XP required per level boundary.
// level = floor(xp / XPPerLevel) + 1
const XPPerLevel = 500

// AddGlobalXP atomically increments global_xp and recomputes global_level.
func (s *Store) AddGlobalXP(ctx context.Context, userID string, xpDelta int) error {
	if xpDelta <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE profiles.users
		SET global_xp    = global_xp + $1,
		    global_level = (global_xp + $1) / $2 + 1,
		    updated_at   = now()
		WHERE user_id = $3
	`, xpDelta, XPPerLevel, userID)
	return err
}

// AddLocalXP atomically increments local_xp and recomputes local_level for a venue profile.
// If the venue profile does not exist, it's a no-op (will be created on next checkin).
func (s *Store) AddLocalXP(ctx context.Context, userID, venueID string, xpDelta int) error {
	if xpDelta <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE profiles.venue_profiles
		SET local_xp    = local_xp + $1,
		    local_level = (local_xp + $1) / $2 + 1
		WHERE user_id = $3 AND venue_id = $4
	`, xpDelta, XPPerLevel, userID, venueID)
	return err
}

// GetUserByNFCUID fetches the user linked to a NiteTap by NFC UID.
// Returns ErrNotFound if the UID is unknown or the tap is anonymous/revoked.
func (s *Store) GetUserByNFCUID(ctx context.Context, nfcUID string) (*User, error) {
	user := &User{}
	var venueID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT u.user_id, u.email, u.display_name, u.role, u.venue_id,
		       u.global_xp, u.global_level, u.created_at, u.updated_at
		FROM profiles.users u
		JOIN profiles.nitetaps t ON t.user_id = u.user_id
		WHERE t.nfc_uid = $1 AND t.status = 'active' AND t.is_anonymous = false
	`, nfcUID).Scan(
		&user.UserID, &user.Email, &user.DisplayName, &user.Role,
		&venueID, &user.GlobalXP, &user.GlobalLevel, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by nfc uid: %w", err)
	}
	user.VenueID = venueID.String
	return user, nil
}

// ── NiteTaps ──────────────────────────────────────────────────────────────────

// GetNiteTap fetches a NiteTap by NFC UID.
func (s *Store) GetNiteTap(ctx context.Context, nfcUID string) (*NiteTap, error) {
	tap := &NiteTap{}
	err := s.db.QueryRowContext(ctx, `
		SELECT tap_id, nfc_uid, user_id, venue_id, is_anonymous, status, issued_at, registered_at, revoked_at
		FROM profiles.nitetaps WHERE nfc_uid = $1
	`, nfcUID).Scan(
		&tap.TapID, &tap.NFCUID, &tap.UserID, &tap.VenueID,
		&tap.IsAnonymous, &tap.Status, &tap.IssuedAt, &tap.RegisteredAt, &tap.RevokedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get nitetap: %w", err)
	}
	return tap, nil
}

// LinkNiteTap associates an anonymous tap with a user account.
func (s *Store) LinkNiteTap(ctx context.Context, nfcUID, userID string) error {
	now := time.Now()
	result, err := s.db.ExecContext(ctx, `
		UPDATE profiles.nitetaps
		SET user_id = $1, is_anonymous = false, registered_at = $2
		WHERE nfc_uid = $3 AND status = 'active' AND (user_id IS NULL OR user_id = $1)
	`, userID, now, nfcUID)
	if err != nil {
		return fmt.Errorf("link nitetap: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateNiteTap creates a new NiteTap record (anonymous or linked).
func (s *Store) CreateNiteTap(ctx context.Context, nfcUID string, userID *string) (*NiteTap, error) {
	tap := &NiteTap{}
	isAnon := userID == nil
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO profiles.nitetaps (nfc_uid, user_id, is_anonymous)
		VALUES ($1, $2, $3)
		RETURNING tap_id, nfc_uid, user_id, venue_id, is_anonymous, status, issued_at, registered_at, revoked_at
	`, nfcUID, userID, isAnon).Scan(
		&tap.TapID, &tap.NFCUID, &tap.UserID, &tap.VenueID,
		&tap.IsAnonymous, &tap.Status, &tap.IssuedAt, &tap.RegisteredAt, &tap.RevokedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create nitetap: %w", err)
	}
	return tap, nil
}

// isUniqueViolation checks for Postgres unique_violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() != "" && contains(err.Error(), "23505") ||
		contains(err.Error(), "duplicate key") ||
		contains(err.Error(), "unique constraint")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
