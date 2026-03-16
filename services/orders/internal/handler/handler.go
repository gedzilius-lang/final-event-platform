package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"niteos.internal/orders/internal/store"
	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
)

type ordersStore interface {
	CreateOrder(ctx context.Context, o *store.Order, items []store.OrderItem) (*store.Order, error)
	FinalizeOrder(ctx context.Context, orderID, ledgerEventID, guestUserID string) (*store.Order, error)
	VoidOrder(ctx context.Context, orderID string) (*store.Order, error)
	GetOrder(ctx context.Context, orderID string) (*store.Order, error)
}

type Handler struct {
	store       ordersStore
	ledgerURL   string
	sessionsURL string
}

func New(s ordersStore, ledgerURL, sessionsURL string) *Handler {
	return &Handler{store: s, ledgerURL: ledgerURL, sessionsURL: sessionsURL}
}

type createOrderReq struct {
	VenueID        string            `json:"venue_id"`
	GuestSessionID string            `json:"guest_session_id"`
	GuestUserID    string            `json:"guest_user_id"`
	Items          []store.OrderItem `json:"items"`
	IdempotencyKey string            `json:"idempotency_key"`
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "bartender" && role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "bartender required")
		return
	}
	var req createOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(req.Items) == 0 || req.IdempotencyKey == "" {
		httputil.RespondError(w, http.StatusBadRequest, "items and idempotency_key required")
		return
	}
	total := 0
	for _, i := range req.Items {
		total += i.PriceNC * i.Quantity
	}
	o := &store.Order{
		VenueID:        req.VenueID,
		DeviceID:       middleware.DeviceID(r.Context()),
		StaffUserID:    middleware.UserID(r.Context()),
		GuestSessionID: req.GuestSessionID,
		TotalNC:        total,
		IdempotencyKey: req.IdempotencyKey,
	}
	out, err := h.store.CreateOrder(r.Context(), o, req.Items)
	if err != nil {
		slog.Error("create order", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusCreated, out)
}

func (h *Handler) FinalizeOrder(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "bartender" && role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}
	orderID := r.PathValue("order_id")
	var req struct {
		GuestUserID string `json:"guest_user_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	ord, err := h.store.GetOrder(r.Context(), orderID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "order not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if ord.Status != "pending" {
		httputil.RespondError(w, http.StatusConflict, "order already "+ord.Status)
		return
	}

	// Check balance
	if req.GuestUserID != "" {
		bal, err := h.fetchUserBalance(r.Context(), req.GuestUserID, ord.VenueID)
		if err != nil {
			slog.Error("balance check", "err", err)
			httputil.RespondError(w, http.StatusInternalServerError, "balance check failed")
			return
		}
		if bal < ord.TotalNC {
			httputil.RespondError(w, http.StatusPaymentRequired, "insufficient balance")
			return
		}
	}

	iKey := fmt.Sprintf("orders:%s:order_paid", orderID)
	event, err := h.writeLedgerEvent(r.Context(), map[string]any{
		"event_type":      "order_paid",
		"user_id":         req.GuestUserID,
		"venue_id":        ord.VenueID,
		"device_id":       ord.DeviceID,
		"amount_nc":       -ord.TotalNC,
		"idempotency_key": iKey,
		"reference_id":    orderID,
	})
	if err != nil {
		slog.Error("write ledger event", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "ledger write failed")
		return
	}

	eventID, _ := event["event_id"].(string)
	out, err := h.store.FinalizeOrder(r.Context(), orderID, eventID, req.GuestUserID)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	go h.incrementSessionSpend(context.Background(), ord.GuestSessionID, ord.TotalNC)

	httputil.Respond(w, http.StatusOK, out)
}

func (h *Handler) VoidOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("order_id")
	ord, err := h.store.GetOrder(r.Context(), orderID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	role := middleware.UserRole(r.Context())
	callerUID := middleware.UserID(r.Context())
	canVoid := role == "nitecore" || role == "venue_admin" ||
		(role == "bartender" && ord.StaffUserID == callerUID && time.Since(ord.CreatedAt) < 2*time.Minute)
	if !canVoid {
		httputil.RespondError(w, http.StatusForbidden, "cannot void")
		return
	}
	prevStatus := ord.Status
	out, err := h.store.VoidOrder(r.Context(), orderID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusConflict, "order not voidable")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// For paid orders: write compensating refund_created event to restore balance.
	if prevStatus == "paid" && ord.GuestUserID != "" {
		go h.writeLedgerEvent(context.Background(), map[string]any{
			"event_type":      "refund_created",
			"user_id":         ord.GuestUserID,
			"venue_id":        ord.VenueID,
			"amount_nc":       ord.TotalNC, // positive: restores balance
			"idempotency_key": fmt.Sprintf("orders:%s:refund_created", orderID),
			"reference_id":    orderID,
		})
	}

	httputil.Respond(w, http.StatusOK, out)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	ord, err := h.store.GetOrder(r.Context(), r.PathValue("order_id"))
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, ord)
}

func (h *Handler) fetchUserBalance(ctx context.Context, userID, venueID string) (int, error) {
	url := fmt.Sprintf("%s/balance/%s/venue/%s", h.ledgerURL, userID, venueID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("X-Internal-Service", "orders")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var result struct {
		BalanceNC int `json:"balance_nc"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.BalanceNC, nil
}

func (h *Handler) writeLedgerEvent(ctx context.Context, payload map[string]any) (map[string]any, error) {
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, h.ledgerURL+"/events", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "orders")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ledger returned %d", resp.StatusCode)
	}
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

func (h *Handler) incrementSessionSpend(ctx context.Context, sessionID string, amountNC int) {
	data, _ := json.Marshal(map[string]int{"amount_nc": amountNC})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		h.sessionsURL+"/"+sessionID+"/spend", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "orders")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("increment session spend", "err", err)
		return
	}
	resp.Body.Close()
}
