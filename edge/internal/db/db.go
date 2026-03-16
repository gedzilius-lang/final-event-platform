// Package db sets up the SQLite database for the edge service.
// Schema mirrors cloud ledger and adds sync_queue and catalog_cache tables.
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open opens (or creates) the edge SQLite database and runs migrations.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite: single writer

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS ledger_events (
		event_id        TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		event_type      TEXT NOT NULL,
		user_id         TEXT NOT NULL,
		venue_id        TEXT,
		device_id       TEXT,
		amount_nc       INTEGER NOT NULL CHECK (amount_nc != 0),
		amount_chf      REAL,
		reference_id    TEXT,
		idempotency_key TEXT NOT NULL UNIQUE,
		occurred_at     DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		synced_from     TEXT NOT NULL DEFAULT 'edge',
		written_by      TEXT NOT NULL,
		synced_at       DATETIME,
		sync_status     TEXT NOT NULL DEFAULT 'pending'
	);

	CREATE TABLE IF NOT EXISTS catalog_items (
		item_id     TEXT PRIMARY KEY,
		venue_id    TEXT NOT NULL,
		name        TEXT NOT NULL,
		price_nc    INTEGER NOT NULL,
		is_active   INTEGER NOT NULL DEFAULT 1,
		category    TEXT,
		updated_at  DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sync_queue (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		idempotency_key TEXT NOT NULL UNIQUE REFERENCES ledger_events(idempotency_key),
		attempts        INTEGER NOT NULL DEFAULT 0,
		last_attempt_at DATETIME,
		status          TEXT NOT NULL DEFAULT 'pending'
	);

	CREATE INDEX IF NOT EXISTS idx_ledger_user ON ledger_events(user_id);
	CREATE INDEX IF NOT EXISTS idx_sync_queue_status ON sync_queue(status);
	`)
	return err
}
