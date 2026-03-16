// Package handler implements the edge LAN API.
// The edge service provides offline-capable order processing and catalog access.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"niteos.internal/edge/internal/catalog"
	"niteos.internal/edge/internal/ledger"
	"niteos.internal/pkg/httputil"
)

type Handler struct {
	ledger  *ledger.Store
	catalog *catalog.Cache
	venueID string
}

func New(l *ledger.Store, c *catalog.Cache, venueID string) *Handler {
	return &Handler{ledger: l, catalog: c, venueID: venueID}
}

// GetBalance handles GET /balance/{user_id}
func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	bal, err := h.ledger.GetBalance(r.Context(), userID)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, map[string]any{
		"user_id":    userID,
		"balance_nc": bal,
	})
}

// CreateOrder handles POST /orders — offline order creation.
// Checks local balance, writes order_paid event to local ledger.
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID         string            `json:"user_id"`
		Items          []orderItem       `json:"items"`
		IdempotencyKey string            `json:"idempotency_key"`
		DeviceID       string            `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.UserID == "" || len(req.Items) == 0 || req.IdempotencyKey == "" {
		httputil.RespondError(w, http.StatusBadRequest, "user_id, items, idempotency_key required")
		return
	}

	// Resolve items from local catalog and compute total
	total := 0
	for i, item := range req.Items {
		catalogItem, err := h.catalog.GetItem(r.Context(), item.CatalogItemID)
		if err != nil {
			httputil.RespondError(w, http.StatusBadRequest, fmt.Sprintf("item %s not found", item.CatalogItemID))
			return
		}
		req.Items[i].Name = catalogItem.Name
		req.Items[i].PriceNC = catalogItem.PriceNC
		total += catalogItem.PriceNC * item.Quantity
	}

	// Check balance
	bal, err := h.ledger.GetBalance(r.Context(), req.UserID)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "balance check failed")
		return
	}
	if bal < total {
		httputil.RespondError(w, http.StatusPaymentRequired, "insufficient balance")
		return
	}

	// Write order_paid event to local ledger
	event, err := h.ledger.WriteEvent(r.Context(), &ledger.Event{
		EventType:      "order_paid",
		UserID:         req.UserID,
		VenueID:        h.venueID,
		DeviceID:       req.DeviceID,
		AmountNC:       -total,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "ledger write failed")
		return
	}

	httputil.Respond(w, http.StatusCreated, map[string]any{
		"event_id":  event.EventID,
		"user_id":   req.UserID,
		"total_nc":  total,
		"items":     req.Items,
		"status":    "paid",
		"synced":    false,
	})
}

// ListItems handles GET /catalog/items — local catalog.
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	items, err := h.catalog.ListItems(r.Context())
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if items == nil {
		items = []*catalog.Item{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"items": items})
}

// SyncStatus handles GET /sync/status
func (h *Handler) SyncStatus(w http.ResponseWriter, r *http.Request) {
	pending, err := h.ledger.PendingSyncEvents(r.Context(), 1000)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, map[string]any{
		"pending_count": len(pending),
	})
}

type orderItem struct {
	CatalogItemID string `json:"catalog_item_id"`
	Name          string `json:"name,omitempty"`
	PriceNC       int    `json:"price_nc,omitempty"`
	Quantity      int    `json:"quantity"`
}
