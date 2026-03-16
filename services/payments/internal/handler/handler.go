// Package handler implements payments service HTTP handlers.
// Topup flow: POST /topup/intent → provider webhook → ledger event → balance increase.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"niteos.internal/payments/internal/domain"
	"niteos.internal/payments/internal/provider"
	"niteos.internal/payments/internal/store"
	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
)

type paymentsStore interface {
	CreateTopup(ctx context.Context, userID, prov, intentID, iKey string, amountCHF float64) (*domain.Topup, error)
	ConfirmTopup(ctx context.Context, intentID string) (*domain.Topup, error)
	GetTopup(ctx context.Context, topupID string) (*domain.Topup, error)
}

type Handler struct {
	store     paymentsStore
	providers map[string]provider.Provider
	ledgerURL string
}

func New(s paymentsStore, providers map[string]provider.Provider, ledgerURL string) *Handler {
	return &Handler{store: s, providers: providers, ledgerURL: ledgerURL}
}

// CreateIntent handles POST /topup/intent.
// Validates input, creates provider payment intent, stores pending topup.
func (h *Handler) CreateIntent(w http.ResponseWriter, r *http.Request) {
	callerUID := middleware.UserID(r.Context())

	var req domain.CreateIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == "" {
		req.UserID = callerUID
	}
	if req.UserID == "" {
		httputil.RespondError(w, http.StatusBadRequest, "user_id required")
		return
	}
	if req.AmountCHF <= 0 || req.AmountCHF > 500 {
		httputil.RespondError(w, http.StatusBadRequest, "amount_chf must be > 0 and <= 500")
		return
	}
	if req.IdempotencyKey == "" {
		httputil.RespondError(w, http.StatusBadRequest, "idempotency_key required")
		return
	}
	if req.Provider == "" {
		req.Provider = domain.ProviderStripe
	}

	prov, ok := h.providers[req.Provider]
	if !ok {
		httputil.RespondError(w, http.StatusBadRequest, "unknown provider: "+req.Provider)
		return
	}

	intent, err := prov.CreateIntent(r.Context(), req.AmountCHF, map[string]string{
		"user_id":         req.UserID,
		"idempotency_key": req.IdempotencyKey,
		"amount_nc":       fmt.Sprintf("%d", domain.CHFToNC(req.AmountCHF)),
	})
	if err != nil {
		slog.Error("create payment intent", "provider", req.Provider, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "payment provider error")
		return
	}

	// Write topup_pending ledger event (informational, excluded from balance)
	go h.writeLedgerEvent(context.Background(), map[string]any{
		"event_type":      "topup_pending",
		"user_id":         req.UserID,
		"amount_nc":       domain.CHFToNC(req.AmountCHF),
		"amount_chf":      req.AmountCHF,
		"idempotency_key": "payments:" + req.IdempotencyKey + ":topup_pending",
		"reference_id":    intent.ProviderIntentID,
	})

	topup, err := h.store.CreateTopup(r.Context(), req.UserID, req.Provider,
		intent.ProviderIntentID, req.IdempotencyKey, req.AmountCHF)
	if err != nil {
		slog.Error("store topup", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httputil.Respond(w, http.StatusCreated, domain.IntentResponse{
		TopupID:      topup.TopupID,
		ClientSecret: intent.ClientSecret,
		RedirectURL:  intent.RedirectURL,
		Provider:     req.Provider,
		AmountCHF:    req.AmountCHF,
		AmountNC:     domain.CHFToNC(req.AmountCHF),
	})
}

// WebhookStripe handles POST /topup/webhook/stripe.
func (h *Handler) WebhookStripe(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, domain.ProviderStripe)
}

// WebhookMock handles POST /topup/webhook/mock — for local testing only.
func (h *Handler) WebhookMock(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, domain.ProviderMock)
}

func (h *Handler) handleWebhook(w http.ResponseWriter, r *http.Request, providerName string) {
	prov, ok := h.providers[providerName]
	if !ok {
		httputil.RespondError(w, http.StatusBadRequest, "unknown provider")
		return
	}

	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB
	if err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "read body")
		return
	}

	sig := r.Header.Get("Stripe-Signature")
	intentID, eventType, err := prov.VerifyWebhook(payload, sig)
	if err != nil {
		slog.Warn("webhook signature invalid", "provider", providerName, "err", err)
		httputil.RespondError(w, http.StatusBadRequest, "invalid webhook signature")
		return
	}

	if intentID == "" || eventType != "payment_intent.succeeded" {
		// Not a success event — acknowledge receipt but take no action
		w.WriteHeader(http.StatusOK)
		return
	}

	topup, err := h.store.ConfirmTopup(r.Context(), intentID)
	if err != nil {
		if err == store.ErrNotFound {
			slog.Warn("webhook for unknown intent", "intent_id", intentID)
			w.WriteHeader(http.StatusOK) // acknowledge; don't retry
			return
		}
		slog.Error("confirm topup", "intent_id", intentID, "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if topup.Status != "confirmed" {
		// Already confirmed — idempotent
		w.WriteHeader(http.StatusOK)
		return
	}

	// Write topup_confirmed ledger event (adds to balance)
	if err := h.writeLedgerEventSync(r.Context(), map[string]any{
		"event_type":      "topup_confirmed",
		"user_id":         topup.UserID,
		"amount_nc":       topup.AmountNC,
		"amount_chf":      topup.AmountCHF,
		"idempotency_key": "payments:" + topup.IdempotencyKey + ":topup_confirmed",
		"reference_id":    topup.TopupID,
		"synced_from":     "cloud",
	}); err != nil {
		slog.Error("write topup_confirmed ledger event", "topup_id", topup.TopupID, "err", err)
		// Return 500 so provider retries the webhook
		httputil.RespondError(w, http.StatusInternalServerError, "ledger write failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetTopup handles GET /topup/{topup_id}.
func (h *Handler) GetTopup(w http.ResponseWriter, r *http.Request) {
	topupID := r.PathValue("topup_id")
	t, err := h.store.GetTopup(r.Context(), topupID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "topup not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Ownership check: only the owner or nitecore can view
	callerUID := middleware.UserID(r.Context())
	isInternal := r.Header.Get("X-Internal-Service") != ""
	if !isInternal && callerUID != t.UserID {
		role := middleware.UserRole(r.Context())
		if role != "nitecore" {
			httputil.RespondError(w, http.StatusForbidden, "forbidden")
			return
		}
	}
	httputil.Respond(w, http.StatusOK, t)
}

func (h *Handler) writeLedgerEvent(ctx context.Context, payload map[string]any) {
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, h.ledgerURL+"/events", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "payments")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("write ledger event (async)", "err", err)
		return
	}
	resp.Body.Close()
}

func (h *Handler) writeLedgerEventSync(ctx context.Context, payload map[string]any) error {
	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.ledgerURL+"/events", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "payments")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("ledger unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ledger returned %d", resp.StatusCode)
	}
	return nil
}
