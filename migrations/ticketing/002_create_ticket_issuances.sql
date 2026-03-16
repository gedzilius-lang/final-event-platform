-- migrations/ticketing/002_create_ticket_issuances.sql
-- Creates the ticketing.ticket_issuances table.
-- Owned by: ticketing service
-- QR token is HMAC-SHA256(secret, issuance_id+user_id+product_id), base64url encoded.

SET search_path = ticketing;

CREATE TABLE IF NOT EXISTS ticket_issuances (
    issuance_id     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      uuid NOT NULL REFERENCES ticketing.ticket_products(product_id),
    user_id         uuid NOT NULL,
    venue_id        uuid NOT NULL,
    qr_token        text UNIQUE NOT NULL,   -- HMAC-signed, single-use scan token
    status          text NOT NULL DEFAULT 'valid'
                      CHECK (status IN ('valid', 'used', 'refunded', 'expired')),
    ledger_event_id uuid NOT NULL,          -- ticket_purchase event that created this
    issued_at       timestamptz NOT NULL DEFAULT now(),
    used_at         timestamptz
);

CREATE INDEX IF NOT EXISTS ticket_issuances_product_idx ON ticket_issuances (product_id);
CREATE INDEX IF NOT EXISTS ticket_issuances_user_idx    ON ticket_issuances (user_id);
CREATE INDEX IF NOT EXISTS ticket_issuances_venue_idx   ON ticket_issuances (venue_id);

GRANT SELECT, INSERT, UPDATE ON ticketing.ticket_issuances TO niteos_app;
