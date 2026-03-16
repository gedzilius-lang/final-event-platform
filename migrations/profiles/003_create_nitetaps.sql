-- migrations/profiles/003_create_nitetaps.sql
-- Creates the profiles.nitetaps table.
-- Owned by: profiles service
-- Replay-attack protected: incrementing counter validated by edge node.

SET search_path = profiles;

CREATE TABLE IF NOT EXISTS nitetaps (
    tap_id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    nfc_uid      text UNIQUE NOT NULL,        -- raw NFC tag UID (hex string)
    user_id      uuid REFERENCES profiles.users(user_id),  -- NULL = anonymous
    venue_id     uuid,                         -- NULL = not venue-scoped
    is_anonymous boolean NOT NULL DEFAULT true,
    issued_at    timestamptz NOT NULL DEFAULT now(),
    registered_at timestamptz,                 -- when user linked it to an account
    revoked_at   timestamptz,
    status       text NOT NULL DEFAULT 'active'
                   CHECK (status IN ('active', 'revoked', 'replaced')),
    metadata     jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS nitetaps_nfc_uid_idx ON nitetaps (nfc_uid);
CREATE INDEX IF NOT EXISTS nitetaps_user_id_idx ON nitetaps (user_id) WHERE user_id IS NOT NULL;

GRANT SELECT, INSERT, UPDATE ON profiles.nitetaps TO niteos_app;
