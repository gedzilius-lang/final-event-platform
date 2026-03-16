// Package domain defines the ledger event types and balance projection logic.
// INVARIANT: amount_nc is never zero. INSERT only — no UPDATE, no DELETE.
package domain

import "time"

// Allowed event types per DATA_OWNERSHIP.md.
const (
	EventTopupPending   = "topup_pending"
	EventTopupConfirmed = "topup_confirmed"
	EventOrderPaid      = "order_paid"
	EventRefundCreated  = "refund_created"
	EventBonusCredit    = "bonus_credit"
	EventTicketPurchase = "ticket_purchase"
	EventVenueCheckin   = "venue_checkin"
	EventSessionClosed  = "session_closed"
	EventMergeAnonymous = "merge_anonymous"
)

// AuthorisedWriters maps event_type to the service allowed to write it.
// Gateway enforces X-Internal-Service header; ledger service double-checks.
var AuthorisedWriters = map[string]string{
	EventTopupPending:   "payments",
	EventTopupConfirmed: "payments",
	EventOrderPaid:      "orders",
	EventRefundCreated:  "payments",
	EventBonusCredit:    "payments",
	EventTicketPurchase: "ticketing",
	EventVenueCheckin:   "sessions",
	EventSessionClosed:  "sessions",
	EventMergeAnonymous: "profiles",
}

// BalanceExcluded lists event types excluded from balance projection.
// These are informational events that don't affect spendable balance.
var BalanceExcluded = map[string]bool{
	EventTopupPending: true,
	EventVenueCheckin: true,
	EventSessionClosed: true,
}

// LedgerEvent is the canonical record written to ledger.ledger_events.
type LedgerEvent struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	UserID         string    `json:"user_id"`
	VenueID        string    `json:"venue_id,omitempty"`
	DeviceID       string    `json:"device_id,omitempty"`
	AmountNC       int       `json:"amount_nc"`
	AmountCHF      *float64  `json:"amount_chf,omitempty"`
	ReferenceID    string    `json:"reference_id,omitempty"`
	IdempotencyKey string    `json:"idempotency_key"`
	OccurredAt     time.Time `json:"occurred_at"`
	SyncedFrom     string    `json:"synced_from"`
	WrittenBy      string    `json:"written_by"`
}

// WriteRequest is the body for POST /events.
type WriteRequest struct {
	EventType      string   `json:"event_type"`
	UserID         string   `json:"user_id"`
	VenueID        string   `json:"venue_id,omitempty"`
	DeviceID       string   `json:"device_id,omitempty"`
	AmountNC       int      `json:"amount_nc"`
	AmountCHF      *float64 `json:"amount_chf,omitempty"`
	ReferenceID    string   `json:"reference_id,omitempty"`
	IdempotencyKey string   `json:"idempotency_key"`
	SyncedFrom     string   `json:"synced_from,omitempty"`
}

// Validate checks all invariants before insertion.
func (r *WriteRequest) Validate() error {
	if r.EventType == "" {
		return ErrInvalidEventType
	}
	if _, ok := AuthorisedWriters[r.EventType]; !ok {
		return ErrInvalidEventType
	}
	if r.UserID == "" {
		return ErrMissingUserID
	}
	if r.AmountNC == 0 {
		return ErrZeroAmount
	}
	if r.IdempotencyKey == "" {
		return ErrMissingIdempotencyKey
	}
	return nil
}

// BalanceResult is returned by GET /balance endpoints.
type BalanceResult struct {
	UserID    string `json:"user_id"`
	VenueID   string `json:"venue_id,omitempty"`
	BalanceNC int    `json:"balance_nc"`
}
