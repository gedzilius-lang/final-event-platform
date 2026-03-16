package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/webhook"
)

// StripeProvider implements Provider using Stripe PaymentIntents.
// Currency is always CHF. Amounts are in Rappen (1 CHF = 100 Rappen).
type StripeProvider struct {
	webhookSecret string
}

func NewStripe(apiKey, webhookSecret string) *StripeProvider {
	stripe.Key = apiKey
	return &StripeProvider{webhookSecret: webhookSecret}
}

func (s *StripeProvider) Name() string { return "stripe" }

func (s *StripeProvider) CreateIntent(_ context.Context, amountCHF float64, metadata map[string]string) (*Intent, error) {
	amountRappen := int64(amountCHF * 100)
	stripeMetadata := make(map[string]string)
	for k, v := range metadata {
		stripeMetadata[k] = v
	}

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountRappen),
		Currency: stripe.String(string(stripe.CurrencyCHF)),
		Metadata: stripeMetadata,
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	if ikey, ok := metadata["idempotency_key"]; ok {
		params.SetIdempotencyKey("stripe_intent_" + ikey)
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe create intent: %w", err)
	}

	return &Intent{
		ProviderIntentID: pi.ID,
		ClientSecret:     pi.ClientSecret,
	}, nil
}

func (s *StripeProvider) VerifyWebhook(payload []byte, signature string) (string, string, error) {
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return "", "", fmt.Errorf("stripe webhook signature invalid: %w", err)
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return "", "", fmt.Errorf("unmarshal payment intent: %w", err)
		}
		return pi.ID, string(event.Type), nil
	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return "", "", fmt.Errorf("unmarshal payment intent: %w", err)
		}
		return pi.ID, string(event.Type), nil
	default:
		// Ignore other event types
		return "", string(event.Type), nil
	}
}
