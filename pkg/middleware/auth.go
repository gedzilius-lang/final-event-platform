// Package middleware provides HTTP middleware for NiteOS services.
// Services trust the X-User-* headers injected by the gateway.
// These headers are NEVER trusted from external clients — only from the gateway.
package middleware

import (
	"context"
	"net/http"
)

type contextKey string

const (
	keyUserID    contextKey = "user_id"
	keyUserRole  contextKey = "user_role"
	keyVenueID   contextKey = "venue_id"
	keyDeviceID  contextKey = "device_id"
	keySessionID contextKey = "session_id"
)

// RequireAuth is HTTP middleware that reads gateway-injected identity headers
// and places them in request context. Returns 401 if X-User-Id is absent.
// Only applied on routes that require authentication.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := r.Header.Get("X-User-Id")
		if uid == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), keyUserID, uid)
		ctx = context.WithValue(ctx, keyUserRole, r.Header.Get("X-User-Role"))
		ctx = context.WithValue(ctx, keyVenueID, r.Header.Get("X-Venue-Id"))
		ctx = context.WithValue(ctx, keyDeviceID, r.Header.Get("X-Device-Id"))
		ctx = context.WithValue(ctx, keySessionID, r.Header.Get("X-Session-Id"))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// WithUserID injects a user ID into context (for testing and internal service calls).
func WithUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, keyUserID, uid)
}

// WithUserRole injects a role into context (for testing and internal service calls).
func WithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, keyUserRole, role)
}

// UserID extracts the authenticated user's ID from context.
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(keyUserID).(string)
	return v
}

// UserRole extracts the authenticated user's role from context.
func UserRole(ctx context.Context) string {
	v, _ := ctx.Value(keyUserRole).(string)
	return v
}

// VenueID extracts the venue_id claim from context (may be empty).
func VenueID(ctx context.Context) string {
	v, _ := ctx.Value(keyVenueID).(string)
	return v
}

// DeviceID extracts the device_id claim from context (may be empty).
func DeviceID(ctx context.Context) string {
	v, _ := ctx.Value(keyDeviceID).(string)
	return v
}

// RequireRole returns 403 if the authenticated role is not in the allowed list.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		set[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := UserRole(r.Context())
			if _, ok := set[role]; !ok {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
