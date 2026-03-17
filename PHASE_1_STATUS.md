# PHASE_1_STATUS.md

Current status of Phase 1 (M0: Safe Ground) execution.

Last updated: 2026-03-17 (guest web complete)

---

## Overall Status: M5 IN PROGRESS — guest web complete, VPS deployment pending

M0–M4 complete (all 13 Go services + edge + admin console). M5 infra artifacts complete. Guest web (web/guest/) implemented and wired into docker-compose.cloud.yml. Remaining M5 blocker: NiteOS VPS deployment (requires GitHub push to allow VPS clone).

---

## Reconciliation: COMPLETE

| Item | Status | Detail |
|------|--------|--------|
| R1: DOMAIN_MODEL.md Venue ownership | FIXED | Removed "sessions service" co-ownership; catalog service is sole owner |
| R2: SYSTEM_ARCHITECTURE.md bonus_credit writer | FIXED | Changed profiles → payments (financial operation) |
| R3: FINAL_REPO_STRUCTURE.md added | DONE | Copied from event-platform-consolidation/docs/ |

---

## M0 Step Status

| Step | Status | Notes |
|------|--------|-------|
| M0.1 Directory skeleton | COMPLETE | All dirs from FINAL_REPO_STRUCTURE.md created; every dir has README.md |
| M0.2 Go workspace | COMPLETE (pending final verification) | go.work + go.mod for 15 modules; stub main.go files; pkg/go.mod now declares jwt/v5 dependency |
| M0.3 docker-compose.dev.yml | COMPLETE (pending final verification) | Postgres + Redis with healthchecks; migrations auto-mounted |
| M0.4 Schema migrations | COMPLETE | migrations/000_init_schemas.sql; 9 schemas + niteos_app role; ledger restriction documented |
| M0.5 Shared tooling | COMPLETE | pkg/jwtutil, pkg/middleware, pkg/httputil, pkg/idempotency; Makefile; .golangci.yml |
| M0.6 RS256 key pair | COMPLETE | Keys in infra/secrets/ (gitignored); docs/SECRET_ROTATION.md complete |
| M0.7 CI skeleton | COMPLETE | ci.yml with go work sync + build + vet + test + lint + migrations dry-run |

---

## Hardening Pass Changes (2026-03-15)

| Fix | File | Reason |
|-----|------|--------|
| Added `require golang-jwt/jwt/v5 v5.2.1` | `pkg/go.mod` | jwtutil.go imports it; go build would fail without declaration |
| Added `go work sync` step before build | `.github/workflows/ci.yml` | Populates go.work.sum checksums before build; prevents module resolution failure |
| Removed `go.work.sum` from .gitignore | `.gitignore` | go.work.sum is a lockfile — must be committed, not ignored |
| Fixed `infra/secrets/` gitignore pattern | `.gitignore` | Changed to `infra/secrets/*` + `!infra/secrets/.gitkeep` so directory is preserved in clones |
| Created `.env.example` | `.env.example` | All env vars documented with dev defaults and production notes |
| Created `docs/LOCAL_DEVELOPMENT.md` | `docs/LOCAL_DEVELOPMENT.md` | Full setup guide: prereqs, key gen, dev stack, commands, troubleshooting |
| Created `M0_ACCEPTANCE.md` | `M0_ACCEPTANCE.md` | Explicit checklist of all M0 exit criteria with pass/fail evidence |
| Created `M0_BLOCKERS.md` | `M0_BLOCKERS.md` | Remaining blockers with exact resolution commands |

---

## Exit Criteria

| Criterion | Status | Action needed |
|-----------|--------|---------------|
| Monorepo exists with full directory structure | PASS | Verified: all 13 services + edge + pkg + web + android + infra + migrations |
| `go work` compiles all service stubs | PENDING | Install Go 1.22+, then: `go work sync && go build ./...` |
| `docker compose up` starts Postgres + Redis | PENDING | Install Docker Desktop, then: `make dev-up` |
| CI pipeline runs and passes | PENDING | Add GitHub remote: `git remote add origin <url> && git push` |
| RS256 key pair generated, rotation procedure documented | PASS | Keys in infra/secrets/; docs/SECRET_ROTATION.md complete |

---

## Blockers

| Blocker | Required for | How to resolve |
|---------|-------------|----------------|
| Go 1.22+ not installed | Verifying M0.2 (compilation) + all of M2/M3 | Install from https://go.dev/dl/ |
| Docker not installed | Verifying M0.3 (dev stack) + all deployment work | Install Docker Desktop |
| No GitHub remote | CI pipeline | `git remote add origin <url> && git push` |

---

## M3 Service Build Status (2026-03-15)

| Service | Status | Notes |
|---------|--------|-------|
| ledger | COMPLETE | Append-only, idempotent, balance projection, domain tests |
| auth | COMPLETE | RS256 JWT, bcrypt, rate limiting, PIN login, JWKS |
| gateway | COMPLETE | JWT validation, Redis revocation, JWKS refresh, reverse proxy |
| profiles | COMPLETE | users, venue_profiles, nitetaps |
| catalog | COMPLETE | venues, items, events, happy hours |
| wallet | COMPLETE | Read-only aggregation over ledger |
| payments | COMPLETE | Stripe + mock provider, topup flow, webhook handler |
| devices | COMPLETE | Enrollment, heartbeat, status management |
| ticketing | COMPLETE | ticket_purchase ledger write, QR validation |
| orders | COMPLETE | Balance check, finalize, void with compensating event, handler tests |
| sessions | COMPLETE | Checkin/checkout, ledger events |
| sync | COMPLETE | Edge frame ingestion, forwarded to cloud ledger |
| reporting | COMPLETE | Venue revenue, session stats, topup summary |
| edge | COMPLETE | SQLite ledger, catalog cache, background sync agent |

## What Was NOT Done (Scope Boundary)

The following belong to later phases and were not started:

- M1: Domain Model Unification (individual service CREATE TABLE migration files)
- M4: Admin or guest web implementation
- M5: VPS A infrastructure changes
- M6: DNS changes or production deployment
- M7: service-2 integration
- TWINT provider (pending Swiss business account credentials)
- No changes to service-1, service-2, or service-3
- No changes to VPS A or VPS B
