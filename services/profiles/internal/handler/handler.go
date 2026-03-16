// Package handler implements the profiles service HTTP handlers.
package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
	"niteos.internal/profiles/internal/store"
)

type profileStore interface {
	CreateUser(ctx context.Context, email, passwordHash, displayName string) (*store.User, error)
	GetUser(ctx context.Context, userID string) (*store.User, error)
	GetUserByEmail(ctx context.Context, email string) (*store.User, error)
	ListUsers(ctx context.Context, emailQuery string) ([]*store.User, error)
	SetUserVenue(ctx context.Context, userID, venueID, role string) error
	GetVenueProfile(ctx context.Context, userID, venueID string) (*store.VenueProfile, error)
	UpsertVenueProfile(ctx context.Context, p store.UpsertVenueProfileParams) (*store.VenueProfile, error)
	GetNiteTap(ctx context.Context, nfcUID string) (*store.NiteTap, error)
	LinkNiteTap(ctx context.Context, nfcUID, userID string) error
	CreateNiteTap(ctx context.Context, nfcUID string, userID *string) (*store.NiteTap, error)
}

// Handler holds dependencies for all profiles endpoints.
type Handler struct {
	store profileStore
}

func New(s profileStore) *Handler {
	return &Handler{store: s}
}

// ── Users ─────────────────────────────────────────────────────────────────────

type createUserRequest struct {
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	DisplayName  string `json:"display_name"`
}

// CreateUser handles POST /users — called by auth service on registration.
// Requires X-Internal-Service: auth header (set by gateway for service-to-service calls).
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Internal-Service") == "" {
		httputil.RespondError(w, http.StatusForbidden, "internal endpoint")
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.PasswordHash == "" {
		httputil.RespondError(w, http.StatusBadRequest, "email and password_hash are required")
		return
	}

	user, err := h.store.CreateUser(r.Context(), req.Email, req.PasswordHash, req.DisplayName)
	if err != nil {
		if err == store.ErrConflict {
			httputil.RespondError(w, http.StatusConflict, "email already registered")
			return
		}
		slog.Error("create user", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.Respond(w, http.StatusCreated, user)
}

// GetUser handles GET /users/{user_id}
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")

	// Requesting own profile or internal service call
	callerUID := middleware.UserID(r.Context())
	isInternal := r.Header.Get("X-Internal-Service") != ""
	if !isInternal && callerUID != userID {
		// nitecore and venue_admin can see any user
		role := middleware.UserRole(r.Context())
		if role != "nitecore" && role != "venue_admin" {
			httputil.RespondError(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "user not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Never expose password_hash
	user.PasswordHash = ""
	httputil.Respond(w, http.StatusOK, user)
}

// GetUserByEmail handles GET /users/by-email/{email} — called by auth service only.
// Returns password_hash so auth can perform bcrypt comparison.
func (h *Handler) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Internal-Service") == "" {
		httputil.RespondError(w, http.StatusForbidden, "internal endpoint")
		return
	}

	email := r.PathValue("email")
	user, err := h.store.GetUserByEmail(r.Context(), email)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "user not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Include password_hash for auth service comparison
	httputil.Respond(w, http.StatusOK, user)
}

// PatchUserVenue handles PATCH /users/{user_id}/venue — nitecore only.
// Sets or clears venue_id and optionally updates role. Used for pilot setup.
func (h *Handler) PatchUserVenue(w http.ResponseWriter, r *http.Request) {
	if middleware.UserRole(r.Context()) != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "nitecore role required")
		return
	}
	userID := r.PathValue("user_id")
	var req struct {
		VenueID string `json:"venue_id"` // empty = clear (NULL)
		Role    string `json:"role"`     // optional; must be valid role if set
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	validRoles := map[string]bool{"guest": true, "venue_admin": true, "bartender": true, "door_staff": true, "nitecore": true}
	if req.Role != "" && !validRoles[req.Role] {
		httputil.RespondError(w, http.StatusBadRequest, "invalid role")
		return
	}
	if err := h.store.SetUserVenue(r.Context(), userID, req.VenueID, req.Role); err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "user not found")
			return
		}
		slog.Error("set user venue", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	user.PasswordHash = ""
	httputil.Respond(w, http.StatusOK, user)
}

// ── NiteTaps ──────────────────────────────────────────────────────────────────

// ListUsers handles GET /users — nitecore only, returns up to 100 users.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "nitecore role required")
		return
	}
	q := r.URL.Query().Get("q")
	users, err := h.store.ListUsers(r.Context(), q)
	if err != nil {
		slog.Error("list users", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if users == nil {
		users = []*store.User{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"users": users, "count": len(users)})
}

// GetNiteTap handles GET /nitetaps/{nfc_uid}
func (h *Handler) GetNiteTap(w http.ResponseWriter, r *http.Request) {
	nfcUID := r.PathValue("nfc_uid")
	tap, err := h.store.GetNiteTap(r.Context(), nfcUID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "tap not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, tap)
}

// LinkNiteTap handles POST /nitetaps/{nfc_uid}/link
func (h *Handler) LinkNiteTap(w http.ResponseWriter, r *http.Request) {
	nfcUID := r.PathValue("nfc_uid")
	userID := middleware.UserID(r.Context())
	if userID == "" {
		httputil.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.store.LinkNiteTap(r.Context(), nfcUID, userID); err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "tap not found or already linked")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, map[string]string{"status": "linked"})
}

// ── VenueProfiles ─────────────────────────────────────────────────────────────

// GetVenueProfile handles GET /venue-profiles/{user_id}/{venue_id}
func (h *Handler) GetVenueProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	venueID := r.PathValue("venue_id")

	vp, err := h.store.GetVenueProfile(r.Context(), userID, venueID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "venue profile not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, vp)
}

type upsertVenueProfileRequest struct {
	UserID  string `json:"user_id"`
	VenueID string `json:"venue_id"`
}

// UpsertVenueProfile handles POST /venue-profiles — called by sessions service on check-in.
func (h *Handler) UpsertVenueProfile(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Internal-Service") == "" {
		httputil.RespondError(w, http.StatusForbidden, "internal endpoint")
		return
	}

	var req upsertVenueProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid request")
		return
	}

	vp, err := h.store.UpsertVenueProfile(r.Context(), store.UpsertVenueProfileParams{
		UserID:  req.UserID,
		VenueID: req.VenueID,
	})
	if err != nil {
		slog.Error("upsert venue profile", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, vp)
}
