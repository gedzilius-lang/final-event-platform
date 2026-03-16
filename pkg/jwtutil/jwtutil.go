// Package jwtutil provides RS256 JWT parsing and validation for NiteOS services.
// Used by: gateway (validation on every request), auth (token issuance).
// No business logic lives here.
package jwtutil

import (
	"crypto/rsa"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the NiteOS JWT payload as defined in SYSTEM_ARCHITECTURE.md.
type Claims struct {
	UID       string `json:"uid"`
	Role      string `json:"role"`
	VenueID   string `json:"venue_id,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	jwt.RegisteredClaims
}

// ValidRole returns true if the role is a recognised NiteOS platform role.
func ValidRole(role string) bool {
	switch role {
	case "guest", "venue_admin", "bartender", "door_staff", "nitecore":
		return true
	}
	return false
}

// Parse validates a signed JWT against the provided RS256 public key.
// Returns the parsed Claims on success.
func Parse(tokenString string, publicKey *rsa.PublicKey) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// IsExpired returns true if the token's expiry time is in the past.
func IsExpired(claims *Claims) bool {
	if claims.ExpiresAt == nil {
		return true
	}
	return claims.ExpiresAt.Time.Before(time.Now())
}
