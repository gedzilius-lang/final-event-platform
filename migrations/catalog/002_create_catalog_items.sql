-- migrations/catalog/002_create_catalog_items.sql
-- Creates the catalog.catalog_items table.
-- Owned by: catalog service
-- Merged from service-1's menu_items + inventory tables.

SET search_path = catalog;

CREATE TABLE IF NOT EXISTS catalog_items (
    item_id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id            uuid NOT NULL REFERENCES catalog.venues(venue_id) ON DELETE CASCADE,
    name                text NOT NULL,
    category            text NOT NULL,
    price_nc            integer NOT NULL CHECK (price_nc > 0),
    icon                text,                        -- emoji or icon identifier
    stock_qty           integer,                     -- NULL = not tracked
    low_threshold       integer DEFAULT 5 CHECK (low_threshold >= 0),
    is_active           boolean NOT NULL DEFAULT true,
    display_order       integer NOT NULL DEFAULT 0,
    happy_hour_price_nc integer CHECK (happy_hour_price_nc > 0),  -- NULL = no override
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS catalog_items_venue_idx    ON catalog_items (venue_id);
CREATE INDEX IF NOT EXISTS catalog_items_category_idx ON catalog_items (venue_id, category);
CREATE INDEX IF NOT EXISTS catalog_items_active_idx   ON catalog_items (venue_id, is_active);

CREATE OR REPLACE FUNCTION catalog.set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS catalog_items_updated_at ON catalog_items;
CREATE TRIGGER catalog_items_updated_at
  BEFORE UPDATE ON catalog_items
  FOR EACH ROW EXECUTE FUNCTION catalog.set_updated_at();

GRANT SELECT, INSERT, UPDATE, DELETE ON catalog.catalog_items TO niteos_app;
