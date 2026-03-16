-- migrations/payments/001_create_payment_intents.sql
-- Creates the payments.payment_intents table.
-- Owned by: payments service
-- Records all fiat payment transactions (TWINT, Stripe).
-- On status=captured, payments service writes topup_confirmed to ledger.

SET search_path = payments;

CREATE TABLE IF NOT EXISTS payment_intents (
    intent_id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL,
    venue_id        uuid NOT NULL,
    provider        text NOT NULL CHECK (provider IN ('twint', 'stripe', 'mock')),
    provider_ref    text,                      -- provider's own reference ID
    amount_chf      numeric(10,2) NOT NULL CHECK (amount_chf > 0),
    status          text NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'confirmed', 'captured', 'refunded', 'failed', 'expired')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    confirmed_at    timestamptz,
    webhook_payload jsonb,                     -- raw verified webhook body
    idempotency_key text UNIQUE NOT NULL
);

CREATE INDEX IF NOT EXISTS payment_intents_user_id_idx      ON payment_intents (user_id);
CREATE INDEX IF NOT EXISTS payment_intents_venue_id_idx     ON payment_intents (venue_id);
CREATE INDEX IF NOT EXISTS payment_intents_status_idx       ON payment_intents (status);
CREATE INDEX IF NOT EXISTS payment_intents_provider_ref_idx ON payment_intents (provider_ref) WHERE provider_ref IS NOT NULL;

-- Grant payments schema to app role
GRANT USAGE ON SCHEMA payments TO niteos_app;
GRANT SELECT, INSERT, UPDATE ON payments.payment_intents TO niteos_app;
