-- migrations/ticketing/001_create_ticket_products.sql
-- Creates the ticketing.ticket_products table.
-- Owned by: ticketing service

SET search_path = ticketing;

CREATE TABLE IF NOT EXISTS ticket_products (
    product_id  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id    uuid NOT NULL,
    event_id    uuid,          -- links to catalog.events (optional, no FK constraint)
    title       text NOT NULL,
    description text,
    price_chf   numeric(10,2) NOT NULL CHECK (price_chf >= 0),
    nc_included integer NOT NULL DEFAULT 0 CHECK (nc_included >= 0),
    capacity    integer CHECK (capacity > 0),  -- NULL = unlimited
    sold_count  integer NOT NULL DEFAULT 0 CHECK (sold_count >= 0),
    status      text NOT NULL DEFAULT 'draft'
                  CHECK (status IN ('draft', 'active', 'sold_out', 'archived')),
    valid_from  timestamptz,
    valid_until timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ticket_products_venue_idx ON ticket_products (venue_id);
CREATE INDEX IF NOT EXISTS ticket_products_event_idx ON ticket_products (event_id) WHERE event_id IS NOT NULL;

GRANT USAGE ON SCHEMA ticketing TO niteos_app;
GRANT SELECT, INSERT, UPDATE ON ticketing.ticket_products TO niteos_app;
