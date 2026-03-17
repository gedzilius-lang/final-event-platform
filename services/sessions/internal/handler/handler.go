package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"niteos.internal/pkg/httputil"
	"niteos.internal/pkg/middleware"
	"niteos.internal/sessions/internal/store"
)

type sessStore interface {
	OpenSession(ctx context.Context, userID, venueID, nfcUID, deviceID string) (*store.VenueSession, error)
	CloseSession(ctx context.Context, sessionID string) (*store.VenueSession, error)
	GetSession(ctx context.Context, sessionID string) (*store.VenueSession, error)
	GetActiveSessionForUser(ctx context.Context, userID string) (*store.VenueSession, error)
	ListActiveSessions(ctx context.Context, venueID string) ([]*store.VenueSession, error)
	IncrementSpend(ctx context.Context, sessionID string, amountNC int) error
}

type Handler struct {
	store          sessStore
	profilesURL    string
	ledgerURL      string
}

func New(s sessStore, profilesURL, ledgerURL string) *Handler {
	return &Handler{store: s, profilesURL: profilesURL, ledgerURL: ledgerURL}
}

func (h *Handler) Checkin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		VenueID  string `json:"venue_id"`
		NfcUID   string `json:"nfc_uid"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.UserID == "" || req.VenueID == "" {
		httputil.RespondError(w, http.StatusBadRequest, "user_id and venue_id required")
		return
	}
	role := middleware.UserRole(r.Context())
	if role != "door_staff" && role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "door_staff required")
		return
	}

	sess, err := h.store.OpenSession(r.Context(), req.UserID, req.VenueID, req.NfcUID, req.DeviceID)
	if err != nil {
		slog.Error("open session", "err", err)
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Write venue_checkin ledger event (fire-and-forget; failure is non-fatal)
	go h.writeLedgerEvent(context.Background(), map[string]any{
		"event_type":      "venue_checkin",
		"user_id":         req.UserID,
		"venue_id":        req.VenueID,
		"amount_nc":       1, // informational only; excluded from balance projection
		"idempotency_key": fmt.Sprintf("sessions:%s:venue_checkin", sess.SessionID),
		"reference_id":    sess.SessionID,
	})

	// Upsert venue profile + award check-in XP (fire-and-forget)
	go h.upsertVenueProfile(context.Background(), req.UserID, req.VenueID)
	go h.awardCheckinXP(context.Background(), req.UserID, req.VenueID)

	httputil.Respond(w, http.StatusCreated, sess)
}

func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	callerUID := middleware.UserID(r.Context())
	sess, err := h.store.GetSession(r.Context(), sessionID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "session not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	// Guest may close own session; door_staff/venue_admin/nitecore may close any
	role := middleware.UserRole(r.Context())
	if sess.UserID != callerUID && role != "door_staff" && role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}

	closed, err := h.store.CloseSession(r.Context(), sessionID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusConflict, "session already closed")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}

	go h.writeLedgerEvent(context.Background(), map[string]any{
		"event_type":      "session_closed",
		"user_id":         closed.UserID,
		"venue_id":        closed.VenueID,
		"amount_nc":       1,
		"idempotency_key": fmt.Sprintf("sessions:%s:session_closed", sessionID),
		"reference_id":    sessionID,
	})

	httputil.Respond(w, http.StatusOK, closed)
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	sess, err := h.store.GetSession(r.Context(), r.PathValue("session_id"))
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "not found")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, sess)
}

// GetGuestSession handles GET /guest/{user_id} — returns the active session for a guest.
// Accessible by the guest themselves or door/security/venue_admin/nitecore.
func (h *Handler) GetGuestSession(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	callerUID := middleware.UserID(r.Context())
	role := middleware.UserRole(r.Context())
	if callerUID != userID &&
		role != "door_staff" && role != "security" &&
		role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}
	sess, err := h.store.GetActiveSessionForUser(r.Context(), userID)
	if err != nil {
		if err == store.ErrNotFound {
			httputil.RespondError(w, http.StatusNotFound, "no active session")
			return
		}
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, sess)
}

func (h *Handler) ListActive(w http.ResponseWriter, r *http.Request) {
	role := middleware.UserRole(r.Context())
	if role != "door_staff" && role != "venue_admin" && role != "nitecore" {
		httputil.RespondError(w, http.StatusForbidden, "forbidden")
		return
	}
	ss, err := h.store.ListActiveSessions(r.Context(), r.PathValue("venue_id"))
	if err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if ss == nil { ss = []*store.VenueSession{} }
	httputil.Respond(w, http.StatusOK, map[string]any{"sessions": ss, "count": len(ss)})
}

func (h *Handler) IncrementSpend(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Internal-Service") == "" {
		httputil.RespondError(w, http.StatusForbidden, "internal endpoint")
		return
	}
	var req struct { AmountNC int `json:"amount_nc"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.store.IncrementSpend(r.Context(), r.PathValue("session_id"), req.AmountNC); err != nil {
		httputil.RespondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httputil.Respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) writeLedgerEvent(ctx context.Context, payload map[string]any) {
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, h.ledgerURL+"/events", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "sessions")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { slog.Error("write ledger event", "err", err); return }
	resp.Body.Close()
}

// awardCheckinXP awards 10 XP for a venue check-in — fire-and-forget.
func (h *Handler) awardCheckinXP(ctx context.Context, userID, venueID string) {
	payload := map[string]any{"xp_delta": 10, "venue_id": venueID}
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		h.profilesURL+"/users/"+userID+"/xp", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "sessions")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("award checkin xp", "err", err)
		return
	}
	resp.Body.Close()
}

func (h *Handler) upsertVenueProfile(ctx context.Context, userID, venueID string) {
	data, _ := json.Marshal(map[string]string{"user_id": userID, "venue_id": venueID})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, h.profilesURL+"/venue-profiles", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Service", "sessions")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { slog.Error("upsert venue profile", "err", err); return }
	resp.Body.Close()
}
