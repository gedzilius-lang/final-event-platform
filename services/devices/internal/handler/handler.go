// Package handler implements devices service HTTP handlers.
package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"niteos.internal/devices/internal/store"
	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
)

type devStore interface {
	Enroll(ctx context.Context, venueID, deviceRole, name string) (*store.Device, error)
	GetDevice(ctx context.Context, deviceID string) (*store.Device, error)
	ListVenueDevices(ctx context.Context, venueID string) ([]*store.Device, error)
	Heartbeat(ctx context.Context, deviceID string) error
	SetStatus(ctx context.Context, deviceID, status string) (*store.Device, error)
}

type Handler struct{ s devStore }

func New(s devStore) *Handler { return &Handler{s: s} }

func (h *Handler) Enroll(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "venue_admin required")
		return
	}
	var req struct {
		VenueID    string `json:"venue_id"`
		DeviceRole string `json:"device_role"`
		Name       string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.VenueID == "" || req.DeviceRole == "" {
		httputil.RespondError(w, http.StatusBadRequest, "venue_id and device_role required")
		return
	}
	d, err := h.s.Enroll(r.Context(), req.VenueID, req.DeviceRole, req.Name)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusCreated, d)
}

func (h *Handler) GetDevice(w http.ResponseWriter, r *http.Request) {
	d, err := h.s.GetDevice(r.Context(), r.PathValue("device_id"))
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "device not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, d)
}

func (h *Handler) ListVenueDevices(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}
	devices, err := h.s.ListVenueDevices(r.Context(), r.PathValue("venue_id"))
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if devices == nil {
		devices = []*store.Device{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"devices": devices})
}

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	deviceID := middleware.DeviceID(r.Context())
	if deviceID == "" {
		deviceID = r.PathValue("device_id")
	}
	if err := h.s.Heartbeat(r.Context(), deviceID); err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "device not found or inactive")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Status != "active" && req.Status != "inactive" && req.Status != "revoked" {
		httputil.RespondError(w, http.StatusBadRequest, "status must be active, inactive, or revoked")
		return
	}
	d, err := h.s.SetStatus(r.Context(), r.PathValue("device_id"), req.Status)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "device not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, d)
}
