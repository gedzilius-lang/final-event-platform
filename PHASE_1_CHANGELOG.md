# PHASE_1_CHANGELOG.md

All changes made during Phase 1 (M0: Safe Ground) of MIGRATION_PLAN.md. Every modification is documented here in order of execution.

---

## Pre-Execution: Document Reconciliation

Before Phase 1 execution began, the following inconsistencies were found between files in this repo and the resolved contradictions documented in `event-platform-consolidation/docs/READINESS_AUDIT.md`.

### R1 — DOMAIN_MODEL.md: Venue ownership corrected
- **File:** `DOMAIN_MODEL.md`
- **Line:** 117
- **Before:** `Owned by: **catalog service** (configuration) + **sessions service** (runtime state)`
- **After:** `Owned by: **catalog service**`
- **Reason:** Sessions service stores venue occupancy in Redis (`venue:{venue_id}:occupancy`) but does not own or write to the `venues` Postgres table. Dual ownership violated the "one service writes" principle in DATA_OWNERSHIP.md. Resolution documented in READINESS_AUDIT.md Finding 1.2.

### R2 — SYSTEM_ARCHITECTURE.md: bonus_credit write authority corrected
- **File:** `SYSTEM_ARCHITECTURE.md`
- **Line:** 412
- **Before:** `- \`profiles\` → \`bonus_credit\` (bonus NC grants)`
- **After:** `- \`payments\` → \`bonus_credit\` (promotional NC grants — financial operation)`
- **Reason:** bonus_credit is a financial event (minting NC). It must flow through the payments service which has the payment context. The profiles service has no authority over NC minting. Resolution documented in READINESS_AUDIT.md Finding 1.1.

### R3 — FINAL_REPO_STRUCTURE.md: Missing file added
- **File:** `FINAL_REPO_STRUCTURE.md`
- **Action:** Copied from `event-platform-consolidation/docs/FINAL_REPO_STRUCTURE.md`
- **Reason:** MIGRATION_PLAN.md M0.1 directly references this file: "Create the full directory skeleton from FINAL_REPO_STRUCTURE.md." It was absent from this repo.

---

## Phase M0 Execution Log

### M0.1 — Directory skeleton created
- Created full directory tree per FINAL_REPO_STRUCTURE.md
- Every directory received a placeholder `README.md` describing what lives there
- Directories: services/×13, edge/, web/guest/, web/admin/, android/×4, infra/, migrations/×9, pkg/×4, scripts/, docs/, .github/workflows/

### M0.2 — Go workspace initialized
- Created `go.work` (Go 1.22)
- Created `go.mod` for all 13 services + edge + pkg (15 modules total)
- Created minimal `cmd/main.go` stub for all 13 services + edge
- All modules: `niteos.internal/{service-name}`
- NOTE: Compilation unverified — Go not installed on this machine. Files are syntactically correct Go 1.22.

### M0.3 — Local development stack
- Created `infra/docker-compose.dev.yml`
- Stack: postgres:16 + redis:7-alpine
- Postgres: database=niteos, user=niteos, password=devpassword (dev only)
- Redis: requirepass devpassword (dev only)
- Ports exposed: 5432, 6379

### M0.4 — Postgres schema provisioning
- Created `migrations/000_init_schemas.sql`
- Schemas created: profiles, ledger, payments, ticketing, orders, catalog, devices, sessions, sync
- Also includes `niteos_app` role with restricted permissions (no UPDATE/DELETE on ledger)
- Migration file naming convention established: `migrations/{schema}/{NNN}_{description}.sql`

### M0.5 — Shared tooling
- Created `pkg/jwtutil/jwtutil.go` — RS256 JWT parsing and validation
- Created `pkg/middleware/auth.go` — HTTP middleware trusting X-User-* headers from gateway
- Created `pkg/httputil/respond.go` — standardized JSON response helpers
- Created `pkg/idempotency/key.go` — idempotency key generation convention
- Created `.golangci.yml` — linter configuration
- Created `Makefile` — targets: build, test, lint, dev-up, dev-down, migrate, seed

### M0.6 — RS256 key pair
- Generated `infra/secrets/jwt_private_key.pem` (2048-bit RSA)
- Generated `infra/secrets/jwt_public_key.pem` (public key extracted)
- `infra/secrets/` added to `.gitignore` — keys never committed
- Created `docs/SECRET_ROTATION.md` — rotation procedure documented

### M0.7 — CI skeleton
- Created `.github/workflows/ci.yml`
- Jobs: go build, go vet, go test, (next build placeholder)
- Runs on: push to main, PR to main

### Supporting files
- Created `.gitignore`
- Created `infra/docker-compose.cloud.yml` (stub — production compose, content filled in M5)
- Created `infra/traefik/traefik.yml` (stub)
- Created `infra/traefik/dynamic/routes.yml` (stub)
- Created `infra/grafana/` directory structure with provisioning stubs
- Created `scripts/dev-up.sh`, `dev-down.sh`, `seed.sh`, `backup.sh`, `migrate.sh`

---

## Exit Criteria Status

| Criterion | Status | Notes |
|-----------|--------|-------|
| Monorepo exists with full directory structure | PASS | All directories and placeholder files created and verified |
| `go work` compiles all service stubs | UNVERIFIED | Go not installed on this machine; files are syntactically correct Go 1.22 |
| `docker compose up` starts Postgres + Redis | UNVERIFIED | Docker not installed on this machine; compose file is correct |
| CI pipeline runs and passes | PENDING | Requires push to GitHub remote |
| RS256 key pair generated and stored | PASS | Generated with openssl 3.5.5; infra/secrets/ gitignored; SECRET_ROTATION.md written |

---

## To verify remaining criteria

```bash
# 1. Install Go 1.22+
# 2. Install Docker Desktop
# 3. Then:
go work sync            # populate go.work.sum checksums
go build ./...          # verifies M0.2
make dev-up             # verifies M0.3
# 4. Push to GitHub to trigger CI (verifies M0.7)
```

---

## Hardening Pass — 2026-03-15

Second pass over all M0 outputs. All defects found are fixed in this pass. No M1 work introduced.

### H1 — pkg/go.mod: missing external dependency declaration
- **File:** `pkg/go.mod`
- **Before:** Module declaration only; no `require` block
- **After:** Added `require github.com/golang-jwt/jwt/v5 v5.2.1`
- **Reason:** `pkg/jwtutil/jwtutil.go` imports `github.com/golang-jwt/jwt/v5`. Without a `require` directive in go.mod, `go build ./pkg/...` fails with "cannot find module providing package". This was a critical compilation blocker.

### H2 — .gitignore: go.work.sum incorrectly excluded
- **File:** `.gitignore`
- **Before:** `go.work.sum` listed under "Go workspace sum (regenerated)"
- **After:** Rule removed; replaced with explanatory comment that go.work.sum is a lockfile that must be committed
- **Reason:** `go.work.sum` contains verified module checksums for workspace builds. Ignoring it means every CI run and every developer must re-verify checksums from the internet, breaking reproducibility. It must be committed alongside `go.work`.

### H3 — .gitignore: infra/secrets/ pattern swallowed .gitkeep
- **File:** `.gitignore`
- **Before:** `infra/secrets/` (ignores entire directory including .gitkeep)
- **After:** `infra/secrets/*` + `!infra/secrets/.gitkeep`
- **Reason:** The old pattern meant `infra/secrets/` would not exist in a fresh clone (git ignores the whole directory, including the .gitkeep placeholder). Developers would get missing-directory errors when running key generation scripts. The new pattern keeps `.gitkeep` tracked (preserving the directory structure) while still ignoring all actual secret files. The `*.pem` and `*.key` rules remain as additional protection.

### H4 — CI: missing go work sync step
- **File:** `.github/workflows/ci.yml`
- **Before:** Build step ran `go build ./...` immediately after setup-go
- **After:** Added `go work sync` step between setup-go and build
- **Reason:** Without `go work sync`, the CI job cannot resolve `github.com/golang-jwt/jwt/v5` checksums (go.work.sum is empty until synced). The build would fail with "missing go.sum entry". `go work sync` fetches checksums from the Go module proxy and writes them to go.work.sum before the build runs.

### H5 — Created .env.example
- **File:** `.env.example` (new)
- **Content:** All NiteOS environment variables with dev defaults and production notes
- **Reason:** Required per M0 acceptance criteria. Provides a safe, committable reference for all environment variables. Dev defaults match `infra/docker-compose.dev.yml`. Production secrets documented as Docker secret file paths, not plaintext values.

### H6 — Created docs/LOCAL_DEVELOPMENT.md
- **File:** `docs/LOCAL_DEVELOPMENT.md` (new)
- **Content:** Complete guide covering prerequisites, first-time setup, dev stack startup, per-service run commands, build/test/lint commands, migration workflow, troubleshooting
- **Reason:** Required per M0 acceptance criteria. Without this, developers have no single reference for getting the local environment running.

### H7 — Created M0_ACCEPTANCE.md
- **File:** `M0_ACCEPTANCE.md` (new)
- **Content:** Explicit pass/fail checklist for all 5 M0 exit criteria with evidence, module table, schema coverage table, reconciliation log
- **Reason:** Required per mission. Provides verifiable record of what was checked and what evidence supports each verdict.

### H8 — Created M0_BLOCKERS.md
- **File:** `M0_BLOCKERS.md` (new)
- **Content:** Three environment blockers (Go, Docker, GitHub remote) with exact resolution commands and expected outcomes
- **Reason:** Required per mission. Provides precise, actionable instructions so the remaining validation steps can be completed with minimal friction as soon as tools are installed.
