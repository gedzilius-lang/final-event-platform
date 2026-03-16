# edge

NiteOS Edge Service. Runs on the Master Tablet (or NiteBox) at each venue.

Provides:
- LAN API on :9000 for NiteKiosk and NiteTerminal devices
- Local SQLite hot ledger (offline-capable)
- Sync agent: flushes event queue to cloud sync service when internet available
- Catalog cache: pulled from cloud catalog service, served locally

See SYSTEM_ARCHITECTURE.md and OFFLINE_SYNC_AND_EDGE_RULES.md for full specification.

This is a stub. Implementation begins in Phase M3.
