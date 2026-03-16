-- migrations/catalog/001_create_venues.sql
-- Creates the catalog.venues table.
-- Owned by: catalog service (sole owner — see DOMAIN_MODEL.md reconciliation R1)
-- sessions service stores occupancy in Redis only; it does NOT own this table.

SET search_path = catalog;

CREATE TABLE IF NOT EXISTS venues (
    venue_id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name           text NOT NULL,
    slug           text UNIQUE NOT NULL,
    city           text NOT NULL DEFAULT 'Zurich',
    address        text,
    capacity       integer NOT NULL DEFAULT 200 CHECK (capacity > 0),
    staff_pin      text NOT NULL,              -- bcrypt hash; used for PIN-based terminal login
    timezone       text NOT NULL DEFAULT 'Europe/Zurich',
    theme          jsonb NOT NULL DEFAULT '{}', -- brand_color, logo_url, font
    stripe_account text,                        -- Stripe Connect account ID (Phase 2)
    is_active      boolean NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS venues_slug_idx ON venues (slug);
CREATE INDEX IF NOT EXISTS venues_active_idx ON venues (is_active);

GRANT USAGE ON SCHEMA catalog TO niteos_app;
GRANT SELECT, INSERT, UPDATE ON catalog.venues TO niteos_app;
