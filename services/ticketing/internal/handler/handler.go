// Package handler implements ticketing service HTTP handlers.
// Ticket purchase deducts from ledger via ticket_purchase event.
package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
	"niteos.internal/ticketing/internal/store"
)

type ticketStore interface {
	IssueTicket(ctx context.Context, userID, eventID, venueID, iKey, qrCode string, priceNC int) (*store.Ticket, error)
	GetTicket(ctx context.Context, ticketID string) (*store.Ticket, error)
	GetTicketByQR(ctx context.Context, qrCode string) (*store.Ticket, error)
	UseTicket(ctx context.Context, ticketID string) (*store.Ticket, error)
	ListUserTickets(ctx context.Context, userID string) ([]*store.Ticket, error)
}

type Handler struct {
	store     ticketStore
	ledgerURL string
}

func New(s ticketStore, ledgerURL string) *Handler {
	return &Handler{store: s, ledgerURL: ledgerURL}
}

type purchaseRequest struct {
	UserID         string `json:"user_id"`
	EventID        string `json:"event_id"`
	VenueID        string `json:"venue_id"`
	PriceNC        int    `json:"price_nc"`
	IdempotencyKey string `json:"idempotency_key"`
}

// PurchaseTicket handles POST /tickets.
// Checks balance via ledger, writes ticket_purchase event, issues ticket.
func (h *Handler) PurchaseTicket(w http.ResponseWriter, r *http.Request) {
	var req purchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	callerUID := middleware.UserID(r.Context())
	if req.UserID == "" {
		req.UserID = callerUID
	}
	if req.UserID == "" || req.EventID == "" || req.VenueID == "" || req.IdempotencyKey == "" {
		httputil.RespondError(w, http.StatusBadRequest, "user_id, event_id, venue_id, idempotency_key required")
		return
	}
	if req.PriceNC <= 0 {
		httputil.RespondError(w, http.StatusBadRequest, "price_nc must be positive")
		return
	}

	// Write ticket_purchase ledger event (idempotent — returns existing on replay)
	event, err := h.writeLedgerEventSync(r.Context(), map[string]any{
		"event_type":      "ticket_purchase",
		"user_id":         req.UserID,
		"venue_id":        req.VenueID,
		"amount_nc":       -req.PriceNC,
		"idempotency_key": "ticketing:" + req.IdempotencyKey + ":ticket_purchase",
		"reference_id":    req.EventID,
	})
	if err != nil {
		slog.Error("ticket_purchase ledger write", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "payment failed")
		return
	}
	_ = event

	qrCode := newQRCode()
	ticket, err := h.store.IssueTicket(r.Context(), req.UserID, req.EventID, req.VenueID,
		req.IdempotencyKey, qrCode, req.PriceNC)
	if err != nil {
		slog.Error("issue ticket", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusCreated, ticket)
}

// GetTicket handles GET /tickets/{ticket_id}
func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	t, err := h.store.GetTicket(r.Context(), r.PathValue("ticket_id"))
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "ticket not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	callerUID := middleware.UserID(r.Context())
	isInternal := r.Header.Get("X-Internal-Service") != ""
	if !isInternal && callerUID != t.UserID {
		role := middleware.UserRole(r.Context())
		if role != "nitecore" && role != "venue_admin" && role != "door_staff" {
			httputil.RespondError(w, http.StatusForbidden, "forbidden")
			return
		}
	}
	httputil.Respond(w, http.StatusOK, t)
}

// ValidateTicket handles POST /tickets/validate — called by sessions service at check-in.
// Marks ticket as used. Returns 409 if already used.
func (h *Handler) ValidateTicket(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	isInternal := r.Header.Get("X-Internal-Service") != ""
	if !isInternal && role != "door_staff" && role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "door_staff required")
		return
	}

	var req struct {
		QRCode string `json:"qr_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.QRCode == "" {
		httputil.RespondError(w, http.StatusBadRequest, "qr_code required")
		return
	}

	t, err := h.store.GetTicketByQR(r.Context(), req.QRCode)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "ticket not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if t.Status != "valid" {
		httputil.RespondError(w, http.StatusConflict, fmt.Sprintf("ticket already %s", t.Status))
		return
	}

	used, err := h.store.UseTicket(r.Context(), t.TicketID)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, used)
}

// ListUserTickets handles GET /tickets/user/{user_id}
func (h *Handler) ListUserTickets(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	callerUID := middleware.UserID(r.Context())
	isInternal := r.Header.Get("X-Internal-Service") != ""
	if !isInternal && callerUID != userID {
		role := middleware.UserRole(r.Context())
		if role != "nitecore" {
			httputil.RespondError(w, http.StatusForbidden, "forbidden")
			return
		}
	}
	tickets, err := h.store.ListUserTickets(r.Context(), userID)
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if tickets == nil {
		tickets = []*store.Ticket{}
	}
	httputil.Respond(w, http.StatusOK, map[string]any{"tickets": tickets})
}

func (h *Handler) writeLedgerEventSync(ctx context.Context, payload map[string]any) (map[string]any, error) {
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, h.ledgerURL+"/events", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "ticketing")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ledger unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ledger returned %d", resp.StatusCode)
	}
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

func newQRCode() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "tix_" + hex.EncodeToString(b)
}
