-- migrations/profiles/001_create_users.sql
-- Creates the profiles.users table.
-- Owned by: profiles service
-- No wallet balance field. Ever. Balance lives in ledger_events only.

SET search_path = profiles;

CREATE TABLE IF NOT EXISTS users (
    user_id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text UNIQUE NOT NULL,
    password_hash text NOT NULL,
    display_name  text,
    role          text NOT NULL DEFAULT 'guest'
                    CHECK (role IN ('guest', 'venue_admin', 'bartender', 'door_staff', 'nitecore')),
    -- venue_id is set for venue-scoped roles (venue_admin, bartender, door_staff).
    -- Null for guest and nitecore roles.
    venue_id      uuid,
    global_xp     integer NOT NULL DEFAULT 0 CHECK (global_xp >= 0),
    global_level  integer NOT NULL DEFAULT 1 CHECK (global_level >= 1),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Index for auth login lookups (by email)
CREATE INDEX IF NOT EXISTS users_email_idx ON users (email);

-- Grant profiles schema to app role
GRANT USAGE ON SCHEMA profiles TO niteos_app;
GRANT SELECT, INSERT, UPDATE ON profiles.users TO niteos_app;

-- updated_at trigger
CREATE OR REPLACE FUNCTION profiles.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_updated_at ON users;
CREATE TRIGGER users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION profiles.set_updated_at();
