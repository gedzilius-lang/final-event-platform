package domain

import "testing"

func TestWriteRequestValidate(t *testing.T) {
	cases := []struct {
		name    string
		req     WriteRequest
		wantErr error
	}{
		{
			"valid topup_confirmed",
			WriteRequest{EventType: EventTopupConfirmed, UserID: "uid", AmountNC: 100, IdempotencyKey: "k"},
			nil,
		},
		{
			"missing event_type",
			WriteRequest{UserID: "uid", AmountNC: 100, IdempotencyKey: "k"},
			ErrInvalidEventType,
		},
		{
			"unknown event_type",
			WriteRequest{EventType: "bogus", UserID: "uid", AmountNC: 100, IdempotencyKey: "k"},
			ErrInvalidEventType,
		},
		{
			"zero amount",
			WriteRequest{EventType: EventOrderPaid, UserID: "uid", AmountNC: 0, IdempotencyKey: "k"},
			ErrZeroAmount,
		},
		{
			"missing user_id",
			WriteRequest{EventType: EventOrderPaid, AmountNC: -10, IdempotencyKey: "k"},
			ErrMissingUserID,
		},
		{
			"missing idempotency_key",
			WriteRequest{EventType: EventOrderPaid, UserID: "uid", AmountNC: -10},
			ErrMissingIdempotencyKey,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantErr != nil && err != tc.wantErr {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestBalanceExclusions(t *testing.T) {
	excluded := []string{EventTopupPending, EventVenueCheckin, EventSessionClosed}
	for _, e := range excluded {
		if !BalanceExcluded[e] {
			t.Errorf("%s should be balance-excluded", e)
		}
	}
	included := []string{EventTopupConfirmed, EventOrderPaid, EventRefundCreated, EventBonusCredit, EventTicketPurchase}
	for _, e := range included {
		if BalanceExcluded[e] {
			t.Errorf("%s should NOT be balance-excluded", e)
		}
	}
}

func TestAuthorisedWriters(t *testing.T) {
	if AuthorisedWriters[EventOrderPaid] != "orders" {
		t.Error("order_paid must be written by orders service")
	}
	if AuthorisedWriters[EventTopupConfirmed] != "payments" {
		t.Error("topup_confirmed must be written by payments service")
	}
	if AuthorisedWriters[EventVenueCheckin] != "sessions" {
		t.Error("venue_checkin must be written by sessions service")
	}
}
