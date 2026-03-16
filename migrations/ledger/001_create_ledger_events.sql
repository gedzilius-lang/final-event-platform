-- migrations/ledger/001_create_ledger_events.sql
-- Creates the ledger.ledger_events table.
-- Owned by: ledger service
-- CRITICAL: INSERT only. No UPDATE. No DELETE. Ever.
-- This is enforced at the Postgres permission level below.

SET search_path = ledger;

CREATE TABLE IF NOT EXISTS ledger_events (
    event_id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type      text NOT NULL
                      CHECK (event_type IN (
                        'topup_pending', 'topup_confirmed', 'order_paid',
                        'refund_created', 'bonus_credit', 'ticket_purchase',
                        'venue_checkin', 'session_closed', 'merge_anonymous'
                      )),
    user_id         uuid NOT NULL,
    venue_id        uuid,
    device_id       uuid,
    amount_nc       integer NOT NULL CHECK (amount_nc <> 0),
    amount_chf      numeric(10,2),
    reference_id    uuid,   -- order_id | payment_intent_id | ticket_issuance_id | session_id
    idempotency_key text UNIQUE NOT NULL,
    occurred_at     timestamptz NOT NULL DEFAULT now(),
    synced_from     text NOT NULL DEFAULT 'cloud',  -- 'cloud' | 'edge:{venue_id}'
    written_by      text NOT NULL,  -- service name: 'payments' | 'orders' | 'sessions' etc.
    metadata        jsonb NOT NULL DEFAULT '{}'
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS ledger_events_user_id_idx     ON ledger_events (user_id);
CREATE INDEX IF NOT EXISTS ledger_events_user_venue_idx  ON ledger_events (user_id, venue_id) WHERE venue_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS ledger_events_venue_id_idx    ON ledger_events (venue_id) WHERE venue_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS ledger_events_occurred_at_idx ON ledger_events (occurred_at DESC);
CREATE INDEX IF NOT EXISTS ledger_events_reference_idx   ON ledger_events (reference_id) WHERE reference_id IS NOT NULL;

-- Grant schema access
GRANT USAGE ON SCHEMA ledger TO niteos_app;

-- Critical: INSERT only — no UPDATE, no DELETE
-- The niteos_app role can write new events but NEVER modify or remove them.
GRANT SELECT, INSERT ON ledger.ledger_events TO niteos_app;
REVOKE UPDATE, DELETE ON ledger.ledger_events FROM niteos_app;

-- Prevent truncation at the table level too
REVOKE TRUNCATE ON ledger.ledger_events FROM niteos_app;
