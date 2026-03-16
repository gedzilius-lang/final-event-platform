-- migrations/orders/001_create_orders.sql
-- Creates the orders.orders and orders.order_items tables.
-- Owned by: orders service
-- order_items stores price/name snapshots — changing catalog does not alter historical orders.

SET search_path = orders;

CREATE TABLE IF NOT EXISTS orders (
    order_id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id         uuid NOT NULL,
    device_id        uuid NOT NULL,      -- which kiosk processed this
    staff_user_id    uuid,               -- which bartender (may be NULL for self-order)
    guest_session_id uuid NOT NULL,      -- references sessions.venue_sessions (no FK — cross-schema)
    total_nc         integer NOT NULL CHECK (total_nc > 0),
    status           text NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'paid', 'voided', 'refunded')),
    ledger_event_id  uuid,               -- order_paid ledger event (set on finalization)
    idempotency_key  text UNIQUE NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    finalized_at     timestamptz
);

CREATE TABLE IF NOT EXISTS order_items (
    item_id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        uuid NOT NULL REFERENCES orders.orders(order_id) ON DELETE CASCADE,
    catalog_item_id uuid NOT NULL,       -- references catalog.catalog_items (no FK — cross-schema)
    name            text NOT NULL,       -- snapshot of item name at order time
    price_nc        integer NOT NULL CHECK (price_nc > 0),  -- snapshot of price at order time
    quantity        integer NOT NULL DEFAULT 1 CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS orders_venue_idx         ON orders (venue_id);
CREATE INDEX IF NOT EXISTS orders_session_idx       ON orders (guest_session_id);
CREATE INDEX IF NOT EXISTS orders_status_idx        ON orders (status);
CREATE INDEX IF NOT EXISTS orders_created_at_idx    ON orders (created_at DESC);
CREATE INDEX IF NOT EXISTS order_items_order_id_idx ON order_items (order_id);

GRANT USAGE ON SCHEMA orders TO niteos_app;
GRANT SELECT, INSERT, UPDATE ON orders.orders      TO niteos_app;
GRANT SELECT, INSERT         ON orders.order_items TO niteos_app;
