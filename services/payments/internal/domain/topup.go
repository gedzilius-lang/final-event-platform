// Package domain defines payment types.
// NC (NiteCoins) are always integer. 1 NC = 1 CHF — fixed peg (PRODUCT_BLUEPRINT §NiteCoin).
package domain

import "time"

// Topup statuses
const (
	StatusPending   = "pending"
	StatusConfirmed = "confirmed"
	StatusFailed    = "failed"
	StatusRefunded  = "refunded"
)

// Provider identifiers
const (
	ProviderStripe = "stripe"
	ProviderTWINT  = "twint"
	ProviderMock   = "mock"
)

// Topup is the canonical payment record.
type Topup struct {
	TopupID          string     `json:"topup_id"`
	UserID           string     `json:"user_id"`
	AmountCHF        float64    `json:"amount_chf"`
	AmountNC         int        `json:"amount_nc"` // = AmountCHF (1:1 peg)
	Provider         string     `json:"provider"`
	ProviderIntentID string     `json:"provider_intent_id"`
	Status           string     `json:"status"`
	IdempotencyKey   string     `json:"idempotency_key"`
	CreatedAt        time.Time  `json:"created_at"`
	ConfirmedAt      *time.Time `json:"confirmed_at,omitempty"`
}

// CHFToNC converts CHF to NiteCoins. 1 CHF = 1 NC (fixed peg, PRODUCT_BLUEPRINT §NiteCoin).
func CHFToNC(chf float64) int {
	return int(chf)
}

// CreateIntentRequest is the body for POST /topup/intent.
type CreateIntentRequest struct {
	UserID         string  `json:"user_id"`
	AmountCHF      float64 `json:"amount_chf"`
	Provider       string  `json:"provider"`
	IdempotencyKey string  `json:"idempotency_key"`
	ReturnURL      string  `json:"return_url,omitempty"` // for redirect-based flows
}

// IntentResponse is returned by POST /topup/intent.
type IntentResponse struct {
	TopupID      string `json:"topup_id"`
	ClientSecret string `json:"client_secret,omitempty"` // Stripe
	RedirectURL  string `json:"redirect_url,omitempty"`  // TWINT
	Provider     string `json:"provider"`
	AmountCHF    float64 `json:"amount_chf"`
	AmountNC     int    `json:"amount_nc"`
}
