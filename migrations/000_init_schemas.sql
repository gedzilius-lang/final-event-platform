-- 000_init_schemas.sql
-- Creates one Postgres schema per NiteOS service.
-- Each service connects to its own schema only and never queries another service's schema directly.
-- Cross-service references are UUID foreign keys stored without FK constraints (application-level validation only).
--
-- Run this migration first, before any service-specific migrations.
-- Migration tool: psql (see scripts/migrate.sh)

-- Service schemas
CREATE SCHEMA IF NOT EXISTS profiles;
CREATE SCHEMA IF NOT EXISTS ledger;
CREATE SCHEMA IF NOT EXISTS payments;
CREATE SCHEMA IF NOT EXISTS ticketing;
CREATE SCHEMA IF NOT EXISTS orders;
CREATE SCHEMA IF NOT EXISTS catalog;
CREATE SCHEMA IF NOT EXISTS devices;
CREATE SCHEMA IF NOT EXISTS sessions;
CREATE SCHEMA IF NOT EXISTS sync;

-- Application role used by all Go services
-- Each service is granted access only to its own schema (granted per-service in service migration files)
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'niteos_app') THEN
    CREATE ROLE niteos_app LOGIN PASSWORD 'changeme_in_production';
  END IF;
END
$$;

-- The ledger schema gets an additional hard constraint:
-- niteos_app may INSERT but never UPDATE or DELETE ledger events.
-- This is enforced at the database level, not only in application code.
-- Grants are applied after the ledger service creates its tables (see migrations/ledger/001_create_ledger_events.sql).
-- Placeholder comment here; actual REVOKE is in the ledger migration.
