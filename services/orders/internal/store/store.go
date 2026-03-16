package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("not found")

type Order struct {
	OrderID        string     `json:"order_id"`
	VenueID        string     `json:"venue_id"`
	DeviceID       string     `json:"device_id"`
	StaffUserID    string     `json:"staff_user_id,omitempty"`
	GuestSessionID string     `json:"guest_session_id"`
	GuestUserID    string     `json:"guest_user_id,omitempty"`
	TotalNC        int        `json:"total_nc"`
	Status         string     `json:"status"`
	LedgerEventID  string     `json:"ledger_event_id,omitempty"`
	IdempotencyKey string     `json:"idempotency_key"`
	CreatedAt      time.Time  `json:"created_at"`
	FinalizedAt    *time.Time `json:"finalized_at,omitempty"`
	Items          []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ItemID        string `json:"item_id"`
	CatalogItemID string `json:"catalog_item_id"`
	Name          string `json:"name"`
	PriceNC       int    `json:"price_nc"`
	Quantity      int    `json:"quantity"`
}

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) CreateOrder(ctx context.Context, o *Order, items []OrderItem) (*Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil { return nil, err }
	defer tx.Rollback()

	out := &Order{}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO orders.orders (venue_id, device_id, staff_user_id, guest_session_id, total_nc, idempotency_key)
		VALUES ($1,$2,NULLIF($3,'')::uuid,$4,$5,$6)
		ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key
		RETURNING order_id, venue_id, device_id, COALESCE(staff_user_id::text,''),
		          guest_session_id, total_nc, status, COALESCE(ledger_event_id::text,''),
		          idempotency_key, created_at, finalized_at
	`, o.VenueID, o.DeviceID, o.StaffUserID, o.GuestSessionID, o.TotalNC, o.IdempotencyKey,
	).Scan(&out.OrderID, &out.VenueID, &out.DeviceID, &out.StaffUserID, &out.GuestSessionID,
		&out.TotalNC, &out.Status, &out.LedgerEventID, &out.IdempotencyKey, &out.CreatedAt, &out.FinalizedAt)
	if err != nil { return nil, fmt.Errorf("create order: %w", err) }

	for _, item := range items {
		var id string
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO orders.order_items (order_id, catalog_item_id, name, price_nc, quantity)
			VALUES ($1,$2,$3,$4,$5)
			RETURNING item_id`,
			out.OrderID, item.CatalogItemID, item.Name, item.PriceNC, item.Quantity,
		).Scan(&id); err != nil { return nil, fmt.Errorf("insert order item: %w", err) }
	}

	if err := tx.Commit(); err != nil { return nil, err }
	out.Items = items
	return out, nil
}

func (s *Store) FinalizeOrder(ctx context.Context, orderID, ledgerEventID, guestUserID string) (*Order, error) {
	now := time.Now()
	out := &Order{}
	err := s.db.QueryRowContext(ctx, `
		UPDATE orders.orders
		SET status='paid', ledger_event_id=$1::uuid,
		    guest_user_id=NULLIF($2,'')::uuid, finalized_at=$3
		WHERE order_id=$4 AND status='pending'
		RETURNING order_id, venue_id, device_id, COALESCE(staff_user_id::text,''),
		          guest_session_id, COALESCE(guest_user_id::text,''), total_nc, status,
		          COALESCE(ledger_event_id::text,''), idempotency_key, created_at, finalized_at
	`, ledgerEventID, guestUserID, now, orderID).Scan(
		&out.OrderID, &out.VenueID, &out.DeviceID, &out.StaffUserID, &out.GuestSessionID,
		&out.GuestUserID, &out.TotalNC, &out.Status, &out.LedgerEventID,
		&out.IdempotencyKey, &out.CreatedAt, &out.FinalizedAt)
	if errors.Is(err, sql.ErrNoRows) { return nil, ErrNotFound }
	if err != nil { return nil, fmt.Errorf("finalize order: %w", err) }
	return out, nil
}

func (s *Store) VoidOrder(ctx context.Context, orderID string) (*Order, error) {
	out := &Order{}
	err := s.db.QueryRowContext(ctx, `
		UPDATE orders.orders SET status='voided'
		WHERE order_id=$1 AND status IN ('pending','paid')
		RETURNING order_id, venue_id, guest_session_id, COALESCE(guest_user_id::text,''),
		          total_nc, status, created_at
	`, orderID).Scan(&out.OrderID, &out.VenueID, &out.GuestSessionID, &out.GuestUserID,
		&out.TotalNC, &out.Status, &out.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) { return nil, ErrNotFound }
	return out, err
}

func (s *Store) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	out := &Order{}
	err := s.db.QueryRowContext(ctx, `
		SELECT order_id, venue_id, device_id, COALESCE(staff_user_id::text,''),
		       guest_session_id, COALESCE(guest_user_id::text,''),
		       total_nc, status, COALESCE(ledger_event_id::text,''),
		       idempotency_key, created_at, finalized_at
		FROM orders.orders WHERE order_id=$1
	`, orderID).Scan(&out.OrderID, &out.VenueID, &out.DeviceID, &out.StaffUserID,
		&out.GuestSessionID, &out.GuestUserID, &out.TotalNC, &out.Status, &out.LedgerEventID,
		&out.IdempotencyKey, &out.CreatedAt, &out.FinalizedAt)
	if errors.Is(err, sql.ErrNoRows) { return nil, ErrNotFound }
	return out, err
}
