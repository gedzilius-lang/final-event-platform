// Package handler implements the sync service HTTP handlers.
// The sync service ingests edge sync frames and writes events to cloud ledger.
// An edge sync frame is a batch of ledger events collected offline by the edge service.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"niteos.internal/pkg/httputil"
)

// SyncEvent is a single event in a sync frame from the edge.
type SyncEvent struct {
	EventType      string   `json:"event_type"`
	UserID         string   `json:"user_id"`
	VenueID        string   `json:"venue_id,omitempty"`
	DeviceID       string   `json:"device_id,omitempty"`
	AmountNC       int      `json:"amount_nc"`
	AmountCHF      *float64 `json:"amount_chf,omitempty"`
	ReferenceID    string   `json:"reference_id,omitempty"`
	IdempotencyKey string   `json:"idempotency_key"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// SyncFrame is a batch of events from an edge device.
type SyncFrame struct {
	DeviceID  string      `json:"device_id"`
	VenueID   string      `json:"venue_id"`
	FrameID   string      `json:"frame_id"`
	Events    []SyncEvent `json:"events"`
	CreatedAt time.Time   `json:"created_at"`
}

// SyncResult reports the outcome of each event in the frame.
type SyncResult struct {
	FrameID  string        `json:"frame_id"`
	Accepted int           `json:"accepted"`
	Rejected int           `json:"rejected"`
	Results  []EventResult `json:"results"`
}

type EventResult struct {
	IdempotencyKey string `json:"idempotency_key"`
	Status         string `json:"status"` // "accepted" | "duplicate" | "rejected"
	Error          string `json:"error,omitempty"`
}

type Handler struct {
	ledgerURL string
}

func New(ledgerURL string) *Handler { return &Handler{ledgerURL: ledgerURL} }

// IngestFrame handles POST /sync/frames.
// Accepts a sync frame from edge, writes all events to cloud ledger.
func (h *Handler) IngestFrame(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Internal-Service") == "" {
		// Edge service must authenticate as internal
		httputil.RespondError(w, http.StatusForbidden, "internal endpoint")
		return
	}

	var frame SyncFrame
	if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
		httputil.RespondError(w, http.StatusBadRequest, "invalid frame")
		return
	}
	if frame.DeviceID == "" || frame.VenueID == "" || len(frame.Events) == 0 {
		httputil.RespondError(w, http.StatusBadRequest, "device_id, venue_id, and events required")
		return
	}

	result := SyncResult{
		FrameID: frame.FrameID,
		Results: make([]EventResult, 0, len(frame.Events)),
	}

	for _, evt := range frame.Events {
		res := h.syncEvent(r.Context(), evt, frame.DeviceID, frame.VenueID)
		result.Results = append(result.Results, res)
		if res.Status == "accepted" || res.Status == "duplicate" {
			result.Accepted++
		} else {
			result.Rejected++
		}
	}

	httputil.Respond(w, http.StatusOK, result)
}

func (h *Handler) syncEvent(ctx context.Context, evt SyncEvent, deviceID, venueID string) EventResult {
	if evt.VenueID == "" {
		evt.VenueID = venueID
	}
	if evt.DeviceID == "" {
		evt.DeviceID = deviceID
	}

	payload := map[string]any{
		"event_type":      evt.EventType,
		"user_id":         evt.UserID,
		"venue_id":        evt.VenueID,
		"device_id":       evt.DeviceID,
		"amount_nc":       evt.AmountNC,
		"amount_chf":      evt.AmountCHF,
		"reference_id":    evt.ReferenceID,
		"idempotency_key": evt.IdempotencyKey,
		"synced_from":     "edge",
	}

	data, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, h.ledgerURL+"/events", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	// Map event_type to the authorised service header
	req.Header.Set("X-Internal-Service", authorisedWriter(evt.EventType))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("sync event to ledger", "ikey", evt.IdempotencyKey, "err", err)
		return EventResult{IdempotencyKey: evt.IdempotencyKey, Status: "rejected", Error: "ledger unreachable"}
	}
	resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return EventResult{IdempotencyKey: evt.IdempotencyKey, Status: "accepted"}
	case http.StatusConflict:
		return EventResult{IdempotencyKey: evt.IdempotencyKey, Status: "duplicate"}
	default:
		return EventResult{IdempotencyKey: evt.IdempotencyKey, Status: "rejected",
			Error: fmt.Sprintf("ledger returned %d", resp.StatusCode)}
	}
}

// authorisedWriter returns the service name that owns the given event type.
func authorisedWriter(eventType string) string {
	m := map[string]string{
		"order_paid":      "orders",
		"session_closed":  "sessions",
		"venue_checkin":   "sessions",
		"ticket_purchase": "ticketing",
		"topup_confirmed": "payments",
		"topup_pending":   "payments",
		"refund_created":  "payments",
		"bonus_credit":    "payments",
		"merge_anonymous": "profiles",
	}
	if s, ok := m[eventType]; ok {
		return s
	}
	return "sync"
}
