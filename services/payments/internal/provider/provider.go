// Package provider defines the payment provider interface and implementations.
// Providers: mock (testing), stripe (production CH), twint (production CH, pending creds).
package provider

import "context"

// Intent is the result of CreateIntent.
type Intent struct {
	ProviderIntentID string
	ClientSecret     string // Stripe: used by frontend SDK
	RedirectURL      string // TWINT: redirect user to this URL
}

// Provider is the payment provider interface.
// All methods must be idempotent with respect to the provider_intent_id.
type Provider interface {
	// CreateIntent creates a new payment intent for the given amount in CHF.
	CreateIntent(ctx context.Context, amountCHF float64, metadata map[string]string) (*Intent, error)

	// VerifyWebhook validates the webhook signature and returns the intent ID and event type.
	// Returns ("", "", err) if the signature is invalid.
	VerifyWebhook(payload []byte, signature string) (intentID, eventType string, err error)

	// Name returns the provider identifier string.
	Name() string
}

// MockProvider is a test provider that auto-confirms without any real payment.
type MockProvider struct{}

func NewMock() *MockProvider { return &MockProvider{} }

func (m *MockProvider) Name() string { return "mock" }

func (m *MockProvider) CreateIntent(_ context.Context, amountCHF float64, metadata map[string]string) (*Intent, error) {
	// Generate a deterministic-ish intent ID from metadata
	uid := metadata["user_id"]
	ikey := metadata["idempotency_key"]
	return &Intent{
		ProviderIntentID: "mock_" + uid[:min(8, len(uid))] + "_" + ikey[:min(8, len(ikey))],
		ClientSecret:     "mock_secret",
	}, nil
}

func (m *MockProvider) VerifyWebhook(payload []byte, _ string) (string, string, error) {
	// Mock: parse intent_id and event_type from a simple JSON payload
	// {"intent_id":"mock_...","event":"payment_intent.succeeded"}
	intentID, eventType := parseMockWebhook(payload)
	return intentID, eventType, nil
}

func parseMockWebhook(payload []byte) (intentID, eventType string) {
	// Minimal JSON parse without encoding/json to avoid import cycle.
	s := string(payload)
	intentID = jsonField(s, "intent_id")
	eventType = jsonField(s, "event")
	return
}

func jsonField(s, key string) string {
	k := `"` + key + `":"`
	idx := indexOf(s, k)
	if idx < 0 {
		return ""
	}
	start := idx + len(k)
	end := indexOf(s[start:], `"`)
	if end < 0 {
		return ""
	}
	return s[start : start+end]
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
