-- migrations/seed_dev.sql
-- Development seed data. NEVER run against production.
-- Creates: one test venue, one nitecore user, one guest user, one bartender,
--          one door_staff user, sample catalog items, and a happy hour rule.
-- Passwords are bcrypt hash of "devpassword" (cost 12).
-- All UUIDs are deterministic for repeatability.

-- ============================================================
-- Profiles: users
-- ============================================================
INSERT INTO profiles.users (user_id, email, password_hash, display_name, role, venue_id)
VALUES
  ('00000000-0000-0000-0000-000000000001',
   'nitecore@niteos.dev',
   '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewohO9T3t2c2P3uy',
   'Nitecore Admin',
   'nitecore', NULL),
  ('00000000-0000-0000-0000-000000000002',
   'admin@venue-alpha.dev',
   '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewohO9T3t2c2P3uy',
   'Venue Alpha Admin',
   'venue_admin', '00000000-0000-0000-0001-000000000001'),
  ('00000000-0000-0000-0000-000000000003',
   'bartender@venue-alpha.dev',
   '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewohO9T3t2c2P3uy',
   'Bar Staff 1',
   'bartender', '00000000-0000-0000-0001-000000000001'),
  ('00000000-0000-0000-0000-000000000004',
   'door@venue-alpha.dev',
   '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewohO9T3t2c2P3uy',
   'Door Staff 1',
   'door_staff', '00000000-0000-0000-0001-000000000001'),
  ('00000000-0000-0000-0000-000000000005',
   'guest@example.dev',
   '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/lewohO9T3t2c2P3uy',
   'Test Guest',
   'guest', NULL)
ON CONFLICT (user_id) DO NOTHING;

-- ============================================================
-- Catalog: venues
-- staff_pin is bcrypt hash of "1234" (cost 12)
-- ============================================================
INSERT INTO catalog.venues (venue_id, name, slug, city, address, capacity, staff_pin, timezone, theme)
VALUES
  ('00000000-0000-0000-0001-000000000001',
   'Venue Alpha',
   'venue-alpha',
   'Zurich',
   'Langstrasse 1, 8004 Zürich',
   300,
   '$2a$12$eImiTXuWVxfM37uY4JANjOgb2WcH2LhkR4HqcsMBdBL0pLkjQGY9.',
   'Europe/Zurich',
   '{"brand_color": "#6C2BD9", "logo_url": "", "font": "Inter"}')
ON CONFLICT (venue_id) DO NOTHING;

-- ============================================================
-- Catalog: catalog_items
-- ============================================================
INSERT INTO catalog.catalog_items (item_id, venue_id, name, category, price_nc, icon, display_order, is_active)
VALUES
  ('00000000-0000-0000-0002-000000000001',
   '00000000-0000-0000-0001-000000000001',
   'Draft Beer', 'drinks', 800, '🍺', 1, true),
  ('00000000-0000-0000-0002-000000000002',
   '00000000-0000-0000-0001-000000000001',
   'Cocktail', 'drinks', 1500, '🍹', 2, true),
  ('00000000-0000-0000-0002-000000000003',
   '00000000-0000-0000-0001-000000000001',
   'Water', 'drinks', 300, '💧', 3, true),
  ('00000000-0000-0000-0002-000000000004',
   '00000000-0000-0000-0001-000000000001',
   'Entry Fee', 'entry', 2000, '🎟️', 10, true)
ON CONFLICT (item_id) DO NOTHING;

-- ============================================================
-- Catalog: happy_hour_rules
-- ============================================================
INSERT INTO catalog.happy_hour_rules (rule_id, venue_id, name, starts_at, ends_at, price_modifier, is_active)
VALUES
  ('00000000-0000-0000-0003-000000000001',
   '00000000-0000-0000-0001-000000000001',
   'Early Bird',
   '20:00',
   '22:00',
   0.80,
   true)
ON CONFLICT (rule_id) DO NOTHING;

-- ============================================================
-- Devices: a test Master Tablet (enrolled, active)
-- ============================================================
INSERT INTO devices.devices (device_id, venue_id, device_role, device_name, public_key, status, enrolled_at)
VALUES
  ('00000000-0000-0000-0004-000000000001',
   '00000000-0000-0000-0001-000000000001',
   'master',
   'Master Tablet - Venue Alpha',
   'dev-placeholder-public-key',
   'active',
   now())
ON CONFLICT (device_id) DO NOTHING;

-- ============================================================
-- Ledger: seed the test guest with 500 NC (5 CHF topup)
-- ============================================================
INSERT INTO ledger.ledger_events
  (event_id, event_type, user_id, venue_id, amount_nc, amount_chf,
   idempotency_key, written_by, synced_from)
VALUES
  ('00000000-0000-0000-0005-000000000001',
   'topup_confirmed',
   '00000000-0000-0000-0000-000000000005',
   '00000000-0000-0000-0001-000000000001',
   50000,
   500.00,
   'seed:guest-initial-topup',
   'payments',
   'cloud')
ON CONFLICT (idempotency_key) DO NOTHING;

-- ============================================================
-- Profiles: venue profile for test guest at venue alpha
-- ============================================================
INSERT INTO profiles.venue_profiles (user_id, venue_id)
VALUES
  ('00000000-0000-0000-0000-000000000005',
   '00000000-0000-0000-0001-000000000001')
ON CONFLICT (user_id, venue_id) DO NOTHING;
