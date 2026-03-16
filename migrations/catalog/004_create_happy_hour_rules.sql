-- migrations/catalog/004_create_happy_hour_rules.sql
-- Creates the catalog.happy_hour_rules table.
-- Owned by: catalog service

SET search_path = catalog;

CREATE TABLE IF NOT EXISTS happy_hour_rules (
    rule_id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id      uuid NOT NULL REFERENCES catalog.venues(venue_id) ON DELETE CASCADE,
    name          text NOT NULL,
    starts_at     time NOT NULL,         -- e.g. 20:00
    ends_at       time NOT NULL,         -- e.g. 22:00
    days_of_week  integer[],             -- 0=Sun...6=Sat; NULL = every day
    price_modifier numeric(4,2) NOT NULL CHECK (price_modifier > 0 AND price_modifier <= 1),
                                         -- e.g. 0.80 = 20% off
    is_active     boolean NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS happy_hour_rules_venue_idx ON happy_hour_rules (venue_id, is_active);

GRANT SELECT, INSERT, UPDATE, DELETE ON catalog.happy_hour_rules TO niteos_app;
