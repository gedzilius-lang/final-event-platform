package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"niteos.internal/catalog/internal/store"
	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
)

type catalogStore interface {
	CreateVenue(ctx context.Context, v *store.Venue, staffPinHash string) (*store.Venue, error)
	GetVenue(ctx context.Context, venueID string) (*store.Venue, error)
	ListVenues(ctx context.Context) ([]*store.Venue, error)
	ListItems(ctx context.Context, venueID string, activeOnly bool) ([]*store.CatalogItem, error)
	GetItem(ctx context.Context, itemID string) (*store.CatalogItem, error)
	CreateItem(ctx context.Context, i *store.CatalogItem) (*store.CatalogItem, error)
	DeleteItem(ctx context.Context, itemID string) error
	ListEvents(ctx context.Context, venueID string) ([]*store.Event, error)
	CreateEvent(ctx context.Context, e *store.Event) (*store.Event, error)
	ListHappyHours(ctx context.Context, venueID string) ([]*store.HappyHourRule, error)
	CreateHappyHour(ctx context.Context, r *store.HappyHourRule) (*store.HappyHourRule, error)
}

type Handler struct{ s catalogStore }

func New(s catalogStore) *Handler { return &Handler{s: s} }

func (h *Handler) CreateVenue(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "nitecore only")
		return
	}
	var req struct {
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		City     string `json:"city"`
		Address  string `json:"address"`
		Capacity int    `json:"capacity"`
		StaffPin string `json:"staff_pin"`
		Timezone string `json:"timezone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.StaffPin == "" {
		httputil.RespondError(w, http.StatusBadRequest, "staff_pin required")
		return
	}
	if req.Timezone == "" {
		req.Timezone = "Europe/Zurich"
	}
	if req.Capacity == 0 {
		req.Capacity = 200
	}
	pinHash, err := bcrypt.GenerateFromPassword([]byte(req.StaffPin), bcrypt.DefaultCost)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	v := &store.Venue{Name: req.Name, Slug: req.Slug, City: req.City, Address: req.Address, Capacity: req.Capacity, Timezone: req.Timezone, Theme: []byte("{}")}
	out, err := h.s.CreateVenue(r.Context(), v, string(pinHash))
	if err != nil {
		if err == store.ErrConflict {
			httputil.RespondError(w, http.StatusConflict, "slug already taken")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusCreated, out)
}

func (h *Handler) GetVenue(w http.ResponseWriter, r *http.Request) {
	v, err := h.s.GetVenue(r.Context(), r.PathValue("venue_id"))
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "venue not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, v)
}

func (h *Handler) UpdateVenue(w http.ResponseWriter, r *http.Request) {
	// Placeholder — full update with partial JSON patch in M4
	httputil.RespondError(w, http.StatusNotImplemented, "not implemented")
}

func (h *Handler) ListVenues(w http.ResponseWriter, r *http.Request) {
	vs, err := h.s.ListVenues(r.Context())
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if vs == nil {
		vs = []*store.Venue{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"venues": vs})
}

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	venueID := r.PathValue("venue_id")
	items, err := h.s.ListItems(r.Context(), venueID, true)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if items == nil {
		items = []*store.CatalogItem{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	item, err := h.s.GetItem(r.Context(), r.PathValue("item_id"))
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "item not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, item)
}

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "venue_admin or nitecore required")
		return
	}
	venueID := r.PathValue("venue_id")
	var req store.CatalogItem
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.VenueID = venueID
	out, err := h.s.CreateItem(r.Context(), &req)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusCreated, out)
}

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	httputil.RespondError(w, http.StatusNotImplemented, "not implemented")
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := h.s.DeleteItem(r.Context(), r.PathValue("item_id")); err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "item not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	evts, err := h.s.ListEvents(r.Context(), r.PathValue("venue_id"))
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if evts == nil {
		evts = []*store.Event{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"events": evts})
}

func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req store.Event
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.VenueID = r.PathValue("venue_id")
	out, err := h.s.CreateEvent(r.Context(), &req)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusCreated, out)
}

func (h *Handler) ListHappyHours(w http.ResponseWriter, r *http.Request) {
	rules, err := h.s.ListHappyHours(r.Context(), r.PathValue("venue_id"))
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if rules == nil {
		rules = []*store.HappyHourRule{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"rules": rules})
}

func (h *Handler) CreateHappyHour(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req store.HappyHourRule
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.VenueID = r.PathValue("venue_id")
	out, err := h.s.CreateHappyHour(r.Context(), &req)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusCreated, out)
}
