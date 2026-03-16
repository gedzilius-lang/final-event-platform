-- migrations/profiles/002_create_venue_profiles.sql
-- Creates the profiles.venue_profiles table.
-- Owned by: profiles service
-- Created on first check-in at a venue.

SET search_path = profiles;

CREATE TABLE IF NOT EXISTS venue_profiles (
    profile_id     uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        uuid NOT NULL REFERENCES profiles.users(user_id) ON DELETE CASCADE,
    venue_id       uuid NOT NULL,  -- references catalog.venues (no FK constraint — cross-schema)
    local_xp       integer NOT NULL DEFAULT 0 CHECK (local_xp >= 0),
    local_level    integer NOT NULL DEFAULT 1 CHECK (local_level >= 1),
    first_visit_at timestamptz,
    last_visit_at  timestamptz,
    visit_count    integer NOT NULL DEFAULT 0 CHECK (visit_count >= 0),
    preferences    jsonb NOT NULL DEFAULT '{}',
    UNIQUE (user_id, venue_id)
);

CREATE INDEX IF NOT EXISTS venue_profiles_user_idx  ON venue_profiles (user_id);
CREATE INDEX IF NOT EXISTS venue_profiles_venue_idx ON venue_profiles (venue_id);

GRANT SELECT, INSERT, UPDATE ON profiles.venue_profiles TO niteos_app;
