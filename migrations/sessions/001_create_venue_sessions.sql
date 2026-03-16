-- migrations/sessions/001_create_venue_sessions.sql
-- Creates the sessions.venue_sessions table.
-- Owned by: sessions service
-- total_spend_nc is a convenience counter updated by orders service.
-- The ledger is still the source of truth for balance.

SET search_path = sessions;

CREATE TABLE IF NOT EXISTS venue_sessions (
    session_id      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL,       -- references profiles.users (no FK — cross-schema)
    venue_id        uuid NOT NULL,       -- references catalog.venues (no FK — cross-schema)
    nitetap_uid     text,                -- NFC UID used to open session (may be anonymous)
    ticket_used     uuid,                -- ticket_issuance_id if entry was via ticket
    opened_at       timestamptz NOT NULL DEFAULT now(),
    closed_at       timestamptz,
    total_spend_nc  integer NOT NULL DEFAULT 0 CHECK (total_spend_nc >= 0),
    checkin_device  uuid,                -- which NiteTerminal scanned them in
    status          text NOT NULL DEFAULT 'open'
                      CHECK (status IN ('open', 'closed'))
);

CREATE INDEX IF NOT EXISTS venue_sessions_user_idx    ON venue_sessions (user_id);
CREATE INDEX IF NOT EXISTS venue_sessions_venue_idx   ON venue_sessions (venue_id);
CREATE INDEX IF NOT EXISTS venue_sessions_status_idx  ON venue_sessions (status);
CREATE INDEX IF NOT EXISTS venue_sessions_tap_idx     ON venue_sessions (nitetap_uid) WHERE nitetap_uid IS NOT NULL;
CREATE INDEX IF NOT EXISTS venue_sessions_opened_idx  ON venue_sessions (opened_at DESC);

GRANT USAGE ON SCHEMA sessions TO niteos_app;
GRANT SELECT, INSERT, UPDATE ON sessions.venue_sessions TO niteos_app;
