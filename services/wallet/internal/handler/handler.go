// Package handler implements wallet service handlers.
// Wallet owns nothing in Postgres — it aggregates from the ledger service.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
)

type Handler struct{ ledgerURL string }

func New(ledgerURL string) *Handler { return &Handler{ledgerURL: ledgerURL} }

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	callerUID := middleware.UserID(r.Context())
	if callerUID != userID {
		role := middleware.UserRole(r.Context())
		if role != "nitecore" && role != "venue_admin" {
			httputil.RespondError(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	balance, err := h.fetchBalance(r.Context(), userID)
	if err != nil {
		httputil.RespondError(w, http.StatusBadGateway, "ledger unavailable")
		return
	}
	httputil.Respond(w, http.StatusOK, map[string]any{
		"user_id":    userID,
		"balance_nc": balance,
	})
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	callerUID := middleware.UserID(r.Context())
	if callerUID != userID {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}

	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")
	if limit == "" { limit = "50" }
	if offset == "" { offset = "0" }

	req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet,
		fmt.Sprintf("%s/events/%s?limit=%s&offset=%s", h.ledgerURL, userID, limit, offset), nil)
	req.Header.Set("X-Internal-Service", "wallet")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		httputil.RespondError(w, http.StatusBadGateway, "ledger unavailable")
		return
	}
	defer resp.Body.Close()

	var result any
	json.NewDecoder(resp.Body).Decode(&result)
	httputil.Respond(w, resp.StatusCode, result)
}

func (h *Handler) fetchBalance(ctx context.Context, userID string) (int, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, h.ledgerURL+"/balance/"+userID, nil)
	req.Header.Set("X-Internal-Service", "wallet")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var result struct {
		BalanceNC int `json:"balance_nc"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.BalanceNC, nil
}
