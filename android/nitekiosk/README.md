# android/nitekiosk

Kotlin Android app. Device Owner (kiosk mode) enforced at OS level.
Credentials stored in Android KeyStore.
LAN-first: connects to edge service at :9000.
Cloud fallback: switches to gateway :8000 if edge unreachable > 3 seconds.

See SYSTEM_ARCHITECTURE.md §Android Applications for full specification.
See FINAL_REPO_STRUCTURE.md for directory layout.

This is a stub. Implementation begins in Phase M3.
