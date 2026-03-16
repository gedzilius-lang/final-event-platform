-- migrations/devices/001_create_devices.sql
-- Creates the devices.devices table.
-- Owned by: devices service
-- Devices generate their own keypair on first boot; public key submitted during enrollment.

SET search_path = devices;

CREATE TABLE IF NOT EXISTS devices (
    device_id      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id       uuid NOT NULL,        -- references catalog.venues (no FK — cross-schema)
    device_role    text NOT NULL
                     CHECK (device_role IN ('kiosk', 'terminal', 'master')),
    device_name    text,                 -- human label: "Bar 1 Kiosk", "Front Door"
    public_key     text NOT NULL,        -- PEM public key for device credential auth
    status         text NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'active', 'revoked')),
    enrolled_at    timestamptz,
    last_heartbeat timestamptz,
    last_seen_ip   text,
    firmware_ver   text,
    metadata       jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS devices_venue_idx  ON devices (venue_id);
CREATE INDEX IF NOT EXISTS devices_status_idx ON devices (status);

GRANT USAGE ON SCHEMA devices TO niteos_app;
GRANT SELECT, INSERT, UPDATE ON devices.devices TO niteos_app;
