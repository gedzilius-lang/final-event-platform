-- migrations/sync/001_create_sync_frames.sql
-- Creates the sync.sync_frames table.
-- Owned by: sync service
-- Edge nodes submit sync frames; the sync service validates and writes to cloud ledger.

SET search_path = sync;

CREATE TABLE IF NOT EXISTS sync_frames (
    frame_id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    venue_id        uuid NOT NULL,
    device_id       uuid NOT NULL,          -- Master Tablet that submitted this frame
    event_count     integer NOT NULL CHECK (event_count > 0),
    event_id_range  text NOT NULL,          -- "{first_event_id}:{last_event_id}"
    checksum        text NOT NULL,          -- SHA256 of serialized events
    events          jsonb NOT NULL,         -- array of LedgerEvent-shaped objects
    submitted_at    timestamptz NOT NULL DEFAULT now(),
    processed_at    timestamptz,
    status          text NOT NULL DEFAULT 'received'
                      CHECK (status IN ('received', 'processing', 'processed', 'failed')),
    failure_reason  text
);

CREATE INDEX IF NOT EXISTS sync_frames_venue_idx  ON sync_frames (venue_id);
CREATE INDEX IF NOT EXISTS sync_frames_status_idx ON sync_frames (status);
CREATE INDEX IF NOT EXISTS sync_frames_sub_idx    ON sync_frames (submitted_at DESC);

GRANT USAGE ON SCHEMA sync TO niteos_app;
GRANT SELECT, INSERT, UPDATE ON sync.sync_frames TO niteos_app;
