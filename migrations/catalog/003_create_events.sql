-- migrations/catalog/003_create_events.sql
-- Creates the catalog.events table (public event listings).
-- Owned by: catalog service
-- NOT to be confused with ledger events or sync events.

SET search_path = catalog;

CREATE TABLE IF NOT EXISTS events (
    event_id   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id   uuid NOT NULL REFERENCES catalog.venues(venue_id) ON DELETE CASCADE,
    title      text NOT NULL,
    starts_at  timestamptz NOT NULL,
    ends_at    timestamptz,
    description text,
    genre      text,
    image_url  text,
    is_public  boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS events_venue_idx      ON events (venue_id);
CREATE INDEX IF NOT EXISTS events_starts_at_idx  ON events (starts_at);
CREATE INDEX IF NOT EXISTS events_public_idx     ON events (is_public, starts_at) WHERE is_public = true;

GRANT SELECT, INSERT, UPDATE, DELETE ON catalog.events TO niteos_app;
