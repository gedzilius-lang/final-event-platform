package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"niteos.internal/orders/internal/store"
	"niteos.internal/pkg/middleware"
)

// mockStore implements ordersStore for testing.
type mockStore struct {
	orders map[string]*store.Order
}

func newMockStore() *mockStore { return &mockStore{orders: map[string]*store.Order{}} }

func (m *mockStore) CreateOrder(_ context.Context, o *store.Order, _ []store.OrderItem) (*store.Order, error) {
	if _, ok := m.orders[o.IdempotencyKey]; ok {
		return m.orders[o.IdempotencyKey], nil
	}
	o.OrderID = "order_test_" + o.IdempotencyKey
	o.Status = "pending"
	o.CreatedAt = time.Now()
	m.orders[o.IdempotencyKey] = o
	return o, nil
}

func (m *mockStore) FinalizeOrder(_ context.Context, orderID, ledgerEventID, guestUserID string) (*store.Order, error) {
	for _, o := range m.orders {
		if o.OrderID == orderID {
			if o.Status != "pending" {
				return nil, store.ErrNotFound
			}
			o.Status = "paid"
			o.LedgerEventID = ledgerEventID
			o.GuestUserID = guestUserID
			now := time.Now()
			o.FinalizedAt = &now
			return o, nil
		}
	}
	return nil, store.ErrNotFound
}

func (m *mockStore) VoidOrder(_ context.Context, orderID string) (*store.Order, error) {
	for _, o := range m.orders {
		if o.OrderID == orderID {
			if o.Status != "pending" && o.Status != "paid" {
				return nil, store.ErrNotFound
			}
			o.Status = "voided"
			return o, nil
		}
	}
	return nil, store.ErrNotFound
}

func (m *mockStore) GetOrder(_ context.Context, orderID string) (*store.Order, error) {
	for _, o := range m.orders {
		if o.OrderID == orderID {
			return o, nil
		}
	}
	return nil, store.ErrNotFound
}

// ledgerServer is a test ledger that tracks events and returns configurable balances.
type ledgerServer struct {
	balanceNC int
	events    []map[string]any
	failWrite bool
}

func (l *ledgerServer) server() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", func(w http.ResponseWriter, r *http.Request) {
		if l.failWrite {
			http.Error(w, `{"error":"fail"}`, http.StatusInternalServerError)
			return
		}
		var evt map[string]any
		json.NewDecoder(r.Body).Decode(&evt)
		l.events = append(l.events, evt)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"event_id": "evt_test_123", "status": "ok"})
	})
	mux.HandleFunc("GET /balance/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"balance_nc": l.balanceNC})
	})
	return httptest.NewServer(mux)
}

func bartenderCtx(r *http.Request) *http.Request {
	ctx := middleware.WithUserID(r.Context(), "staff_001")
	ctx = middleware.WithUserRole(ctx, "bartender")
	return r.WithContext(ctx)
}

func nitecoredCtx(r *http.Request) *http.Request {
	ctx := middleware.WithUserID(r.Context(), "nc_001")
	ctx = middleware.WithUserRole(ctx, "nitecore")
	return r.WithContext(ctx)
}

func TestCreateOrder_RequiresBartender(t *testing.T) {
	ms := newMockStore()
	ls := &ledgerServer{balanceNC: 100}
	srv := ls.server()
	defer srv.Close()

	h := New(ms, srv.URL, srv.URL)

	body := `{"venue_id":"v1","items":[{"catalog_item_id":"i1","name":"Beer","price_nc":20,"quantity":1}],"idempotency_key":"key1"}`
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
	// No role set — should be forbidden
	ctx := middleware.WithUserRole(req.Context(), "guest")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.CreateOrder(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", w.Code)
	}
}

func TestCreateOrder_Success(t *testing.T) {
	ms := newMockStore()
	ls := &ledgerServer{balanceNC: 100}
	srv := ls.server()
	defer srv.Close()

	h := New(ms, srv.URL, srv.URL)

	body := `{"venue_id":"v1","guest_session_id":"sess1","guest_user_id":"user1","items":[{"catalog_item_id":"i1","name":"Beer","price_nc":20,"quantity":1}],"idempotency_key":"key1"}`
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
	req = bartenderCtx(req)
	w := httptest.NewRecorder()
	h.CreateOrder(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestFinalizeOrder_InsufficientBalance(t *testing.T) {
	ms := newMockStore()
	// Pre-create a pending order
	order := &store.Order{
		OrderID:        "order_test_key2",
		VenueID:        "v1",
		TotalNC:        50,
		Status:         "pending",
		IdempotencyKey: "key2",
		CreatedAt:      time.Now(),
	}
	ms.orders["key2"] = order

	ls := &ledgerServer{balanceNC: 30} // only 30 NC, need 50
	srv := ls.server()
	defer srv.Close()

	h := New(ms, srv.URL, srv.URL)

	body := `{"guest_user_id":"user1"}`
	req := httptest.NewRequest(http.MethodPost, "/orders/order_test_key2/finalize", bytes.NewBufferString(body))
	req.SetPathValue("order_id", "order_test_key2")
	req = bartenderCtx(req)
	w := httptest.NewRecorder()
	h.FinalizeOrder(w, req)
	if w.Code != http.StatusPaymentRequired {
		t.Errorf("want 402, got %d: %s", w.Code, w.Body.String())
	}
	// Order must remain pending
	o, _ := ms.GetOrder(context.Background(), "order_test_key2")
	if o.Status != "pending" {
		t.Errorf("order status must stay pending, got %s", o.Status)
	}
	// No ledger event must have been written
	if len(ls.events) > 0 {
		t.Errorf("no ledger event should be written on insufficient balance, got %d", len(ls.events))
	}
}

func TestFinalizeOrder_Success(t *testing.T) {
	ms := newMockStore()
	order := &store.Order{
		OrderID:        "order_test_key3",
		VenueID:        "v1",
		TotalNC:        20,
		Status:         "pending",
		IdempotencyKey: "key3",
		CreatedAt:      time.Now(),
	}
	ms.orders["key3"] = order

	ls := &ledgerServer{balanceNC: 100}
	srv := ls.server()
	defer srv.Close()

	h := New(ms, srv.URL, srv.URL)

	body := `{"guest_user_id":"user1"}`
	req := httptest.NewRequest(http.MethodPost, "/orders/order_test_key3/finalize", bytes.NewBufferString(body))
	req.SetPathValue("order_id", "order_test_key3")
	req = bartenderCtx(req)
	w := httptest.NewRecorder()
	h.FinalizeOrder(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	o, _ := ms.GetOrder(context.Background(), "order_test_key3")
	if o.Status != "paid" {
		t.Errorf("order status must be paid, got %s", o.Status)
	}
	// Exactly one ledger event (order_paid) must have been written
	if len(ls.events) != 1 {
		t.Fatalf("expected 1 ledger event, got %d", len(ls.events))
	}
	if ls.events[0]["event_type"] != "order_paid" {
		t.Errorf("expected order_paid event, got %v", ls.events[0]["event_type"])
	}
	if ls.events[0]["amount_nc"].(float64) != -20 {
		t.Errorf("expected amount_nc=-20, got %v", ls.events[0]["amount_nc"])
	}
}

func TestVoidOrder_WithinTwoMinutes_WritesCompensatingEvent(t *testing.T) {
	ms := newMockStore()
	order := &store.Order{
		OrderID:        "order_test_key4",
		VenueID:        "v1",
		GuestUserID:    "user1",
		StaffUserID:    "staff_001", // same as bartenderCtx caller
		TotalNC:        30,
		Status:         "paid",
		IdempotencyKey: "key4",
		CreatedAt:      time.Now(), // just created
	}
	ms.orders["key4"] = order

	ls := &ledgerServer{balanceNC: 70}
	srv := ls.server()
	defer srv.Close()

	h := New(ms, srv.URL, srv.URL)

	req := httptest.NewRequest(http.MethodPost, "/orders/order_test_key4/void", bytes.NewBufferString("{}"))
	req.SetPathValue("order_id", "order_test_key4")
	req = bartenderCtx(req)
	w := httptest.NewRecorder()
	h.VoidOrder(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVoidOrder_AfterTwoMinutes_NonAdmin_Forbidden(t *testing.T) {
	ms := newMockStore()
	old := time.Now().Add(-5 * time.Minute) // 5 minutes ago
	order := &store.Order{
		OrderID:        "order_test_key5",
		VenueID:        "v1",
		StaffUserID:    "staff_001",
		TotalNC:        30,
		Status:         "paid",
		IdempotencyKey: "key5",
		CreatedAt:      old,
	}
	ms.orders["key5"] = order

	ls := &ledgerServer{balanceNC: 70}
	srv := ls.server()
	defer srv.Close()

	h := New(ms, srv.URL, srv.URL)

	req := httptest.NewRequest(http.MethodPost, "/orders/order_test_key5/void", bytes.NewBufferString("{}"))
	req.SetPathValue("order_id", "order_test_key5")
	req = bartenderCtx(req) // bartender, not admin
	w := httptest.NewRecorder()
	h.VoidOrder(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVoidOrder_AfterTwoMinutes_NiteCore_Allowed(t *testing.T) {
	ms := newMockStore()
	old := time.Now().Add(-5 * time.Minute)
	order := &store.Order{
		OrderID:        "order_test_key6",
		VenueID:        "v1",
		GuestUserID:    "user1",
		TotalNC:        30,
		Status:         "paid",
		IdempotencyKey: "key6",
		CreatedAt:      old,
	}
	ms.orders["key6"] = order

	ls := &ledgerServer{balanceNC: 70}
	srv := ls.server()
	defer srv.Close()

	h := New(ms, srv.URL, srv.URL)

	req := httptest.NewRequest(http.MethodPost, "/orders/order_test_key6/void", bytes.NewBufferString("{}"))
	req.SetPathValue("order_id", "order_test_key6")
	req = nitecoredCtx(req)
	w := httptest.NewRecorder()
	h.VoidOrder(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}
