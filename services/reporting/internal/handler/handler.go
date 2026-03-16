// Package handler implements reporting service HTTP handlers.
// All endpoints are read-only. Nitecore or venue_admin access only.
package handler

import (
	"context"
	"net/http"
	"time"

	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
	"niteos.internal/reporting/internal/store"
)

type reportStore interface {
	GetVenueRevenue(ctx context.Context, venueID string, from, to time.Time) (*store.VenueRevenue, error)
	GetVenueSessionStats(ctx context.Context, venueID string, from, to time.Time) (*store.SessionStats, error)
	GetTopupSummary(ctx context.Context, from, to time.Time) (*store.TopupSummary, error)
}

type Handler struct{ s reportStore }

func New(s reportStore) *Handler { return &Handler{s: s} }

// GetVenueRevenue handles GET /reports/venues/{venue_id}/revenue?from=&to=
func (h *Handler) GetVenueRevenue(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "nitecore" && role != "venue_admin" {
		httputil.RespondError(w, http.StatusForbidden, "nitecore or venue_admin required")
		return
	}
	venueID := r.PathValue("venue_id")
	from, to, err := parseDateRange(r)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.s.GetVenueRevenue(r.Context(), venueID, from, to)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, result)
}

// GetVenueSessions handles GET /reports/venues/{venue_id}/sessions?from=&to=
func (h *Handler) GetVenueSessions(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "nitecore" && role != "venue_admin" {
		httputil.RespondError(w, http.StatusForbidden, "nitecore or venue_admin required")
		return
	}
	venueID := r.PathValue("venue_id")
	from, to, err := parseDateRange(r)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.s.GetVenueSessionStats(r.Context(), venueID, from, to)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, result)
}

// GetTopupSummary handles GET /reports/topups?from=&to=
func (h *Handler) GetTopupSummary(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "nitecore required")
		return
	}
	from, to, err := parseDateRange(r)
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.s.GetTopupSummary(r.Context(), from, to)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, result)
}

func parseDateRange(r *http.Request) (from, to time.Time, err error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		now := time.Now().UTC()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		to = now
		return
	}
	from, err = time.Parse("2006-01-02", fromStr)
	if err != nil {
		return from, to, &dateParseError{"from: " + err.Error()}
	}
	to, err = time.Parse("2006-01-02", toStr)
	if err != nil {
		return from, to, &dateParseError{"to: " + err.Error()}
	}
	to = to.Add(24 * time.Hour) // inclusive end date
	return
}

type dateParseError struct{ msg string }

func (e *dateParseError) Error() string { return "invalid date: " + e.msg }
