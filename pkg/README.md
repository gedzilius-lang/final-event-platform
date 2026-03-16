# pkg — Shared Go Module

Module: `niteos.internal/pkg`
Used by: all 13 cloud services and the edge service.

Contents:
- `jwtutil/`     — RS256 JWT parsing and validation (gateway + auth only for signing)
- `middleware/`  — HTTP middleware trusting X-User-* gateway headers
- `httputil/`    — Standardised JSON response helpers
- `idempotency/` — Idempotency key generation convention

Rules:
- No business logic here.
- No dependency on any service.
- Services depend on pkg; pkg never depends on services.
