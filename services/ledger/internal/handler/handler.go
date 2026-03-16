// Package handler implements ledger service HTTP handlers.
// All writes are append-only. Caller must supply idempotency_key.
// Write authority is enforced via X-Internal-Service header.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"niteos.internal/ledger/internal/domain"
	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
)

type ledgerStore interface {
	WriteEvent(ctx context.Context, req *domain.WriteRequest, writtenBy string) (*domain.LedgerEvent, error)
	GetBalance(ctx context.Context, userID, venueID string) (int, error)
	ListUserEvents(ctx context.Context, userID string, limit, offset int) ([]*domain.LedgerEvent, error)
	ListVenueEvents(ctx context.Context, venueID string, limit, offset int) ([]*domain.LedgerEvent, error)
}

type Handler struct{ store ledgerStore }

func New(s ledgerStore) *Handler { return &Handler{store: s} }

// WriteEvent handles POST /events.
// Enforces: caller identity, event_type→service mapping, non-zero amount, idempotency key.
// Returns 200 (not 201) on idempotent replay so callers can distinguish new vs existing.
func (h *Handler) WriteEvent(w http.ResponseWriter, r *http.Request) {
	caller := r.Header.Get("X-Internal-Service")
	if caller == "" {
		httputil.RespondError(w, http.StatusForbidden, "internal endpoint")
		return
	}

	var req domain.WriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Enforce write authority
	allowed, ok := domain.AuthorisedWriters[req.EventType]
	if !ok || allowed != caller {
		slog.Warn("unauthorised ledger write attempt",
			"caller", caller, "event_type", req.EventType, "allowed", allowed)
		httputil.RespondError(w, http.StatusForbidden, domain.ErrUnauthorisedWriter.Error())
		return
	}

	if req.SyncedFrom == "" {
		req.SyncedFrom = "cloud"
	}

	event, err := h.store.WriteEvent(r.Context(), &req, caller)
	if err != nil {
		slog.Error("write ledger event", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.Respond(w, http.StatusOK, event)
}

// GetBalance handles GET /balance/{user_id}
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	callerUID := middleware.UserID(r.Context())
	isInternal := r.Header.Get("X-Internal-Service") != ""
	if !isInternal && callerUID != userID {
		role := middleware.UserRole(r.Context())
		if role != "nitecore" && role != "venue_admin" {
			httputil.RespondError(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	bal, err := h.store.GetBalance(r.Context(), userID, "")
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, domain.BalanceResult{UserID: userID, BalanceNC: bal})
}

// GetBalanceForVenue handles GET /balance/{user_id}/venue/{venue_id}
func (h *Handler) GetBalanceForVenue(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	venueID := r.PathValue("venue_id")

	bal, err := h.store.GetBalance(r.Context(), userID, venueID)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, domain.BalanceResult{UserID: userID, VenueID: venueID, BalanceNC: bal})
}

// GetUserEvents handles GET /events/{user_id}
func (h *Handler) GetUserEvents(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	limit, offset := pagination(r)

	callerUID := middleware.UserID(r.Context())
	isInternal := r.Header.Get("X-Internal-Service") != ""
	if !isInternal && callerUID != userID {
		role := middleware.UserRole(r.Context())
		if role != "nitecore" && role != "venue_admin" {
			httputil.RespondError(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	events, err := h.store.ListUserEvents(r.Context(), userID, limit, offset)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if events == nil {
		events = []*domain.LedgerEvent{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"events": events, "limit": limit, "offset": offset})
}

// GetVenueEvents handles GET /events/venue/{venue_id}
func (h *Handler) GetVenueEvents(w http.ResponseWriter, r *http.Request) {
	venueID := r.PathValue("venue_id")
	limit, offset := pagination(r)

	role := middleware.UserRole(r.Context())
	isInternal := r.Header.Get("X-Internal-Service") != ""
	if !isInternal && role != "nitecore" && role != "venue_admin" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}

	events, err := h.store.ListVenueEvents(r.Context(), venueID, limit, offset)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if events == nil {
		events = []*domain.LedgerEvent{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"events": events, "limit": limit, "offset": offset})
}

func pagination(r *http.Request) (limit, offset int) {
	limit = 50
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

var _ = errors.New // keep errors import used
