# MIGRATION_PLAN.md

The complete migration from the current three-repo state to the unified NiteOS system. Every phase is independently executable, independently rollbackable, and leaves the existing running systems intact until the final cutover.

The rule governing this plan: **do not break what is working while building what is new.**

service-2 (Market) and service-3 (Radio) are live products. Nothing in this plan ever touches them until the explicit integration phases, and even then integration is additive only. service-1 is a prototype — not in production use — so its migration is lower risk, but the data model it implies informs almost everything.

---

## Current State Inventory

| System | Repo | VPS | Status | Users / Live Traffic |
|--------|------|-----|--------|----------------------|
| service-1 (Fastify API + Next.js OS + Next.js Admin) | `repos/service-1` | VPS A (31.97.126.86) | Prototype. Development only. | No production users. No live transactions. |
| service-2 (People We Like Market) | `repos/service-2` | VPS B (72.60.181.89) | Production v0.4.3 | Live marketplace users. |
| service-3 (Radio) | `repos/service-3` | VPS B (72.60.181.89) | Working | Live listeners. |

---

## Migration Phases Overview

| Phase | Name | What moves | Touches live systems? |
|-------|------|-----------|----------------------|
| M0 | Safe Ground | Repository + tooling setup | No |
| M1 | Domain Model Unification | Schemas, entity ownership, data contracts | No (design only) |
| M2 | Authentication Unification | Replace two auth models with one | service-1 only (not production) |
| M3 | Service Build and Merge | Go microservices, one kernel at a time | No (new system, parallel) |
| M4 | Admin Unification | Unified admin console replaces service-1 admin | service-1 only |
| M5 | Infrastructure Consolidation | Traefik, Docker Compose, VPS A rebuild | VPS A rebuild (service-1 is prototype, acceptable) |
| M6 | Cutover and First Pilot | DNS switch, first live venue | DNS only; service-1 parked |
| M7 | service-2 Auth Integration | Shared identity across Market and NiteOS | service-2 additive only |

---

## Phase M0: Safe Ground

**Goal:** Create the new repository and all shared scaffolding without touching any running system.

**Entry criteria:** None. This phase starts immediately.

**What happens:**

### M0.1 — Create the NiteOS monorepo

```bash
mkdir niteos
cd niteos
git init
git commit --allow-empty -m "init: niteos monorepo"
```

Create the full directory skeleton from FINAL_REPO_STRUCTURE.md. Every directory gets a placeholder `README.md` describing what will live there. No implementation yet.

### M0.2 — Go workspace

```bash
go work init
# for each service: go mod init niteos.internal/{service-name}
go work use ./services/gateway ./services/auth ./services/profiles \
  ./services/ledger ./services/wallet ./services/payments \
  ./services/ticketing ./services/orders ./services/catalog \
  ./services/devices ./services/sessions ./services/reporting \
  ./services/sync ./pkg ./edge
```

Commit. The workspace compiles with empty `cmd/main.go` stubs in every service.

### M0.3 — Local development stack

Write `infra/docker-compose.dev.yml`:

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: niteos
      POSTGRES_USER: niteos
      POSTGRES_PASSWORD: devpassword
    ports: ["5432:5432"]
    volumes: ["pgdata:/var/lib/postgresql/data"]

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass devpassword
    ports: ["6379:6379"]

volumes:
  pgdata:
```

This stack is for development only. VPS A continues to run service-1 unchanged.

### M0.4 — Postgres schema provisioning

Write and run initial migration files. One schema per service:

```sql
-- migrations/000_init_schemas.sql
CREATE SCHEMA IF NOT EXISTS profiles;
CREATE SCHEMA IF NOT EXISTS ledger;
CREATE SCHEMA IF NOT EXISTS payments;
CREATE SCHEMA IF NOT EXISTS ticketing;
CREATE SCHEMA IF NOT EXISTS orders;
CREATE SCHEMA IF NOT EXISTS catalog;
CREATE SCHEMA IF NOT EXISTS devices;
CREATE SCHEMA IF NOT EXISTS sessions;
CREATE SCHEMA IF NOT EXISTS sync;
```

All subsequent migration files are namespaced: `migrations/profiles/001_create_users.sql`, etc.

### M0.5 — Shared tooling

- `pkg/jwtutil` — RS256 JWT parsing and validation (no business logic)
- `pkg/middleware` — HTTP middleware that reads X-User-Id, X-User-Role headers
- `pkg/httputil` — Standardized JSON response helpers
- `pkg/idempotency` — Idempotency key generation convention
- `.golangci.yml` — Linter configuration
- `Makefile` with targets: `build`, `test`, `lint`, `dev-up`, `dev-down`, `migrate`, `seed`

### M0.6 — RS256 key pair

```bash
openssl genrsa -out jwt_private_key.pem 2048
openssl rsa -in jwt_private_key.pem -pubout -out jwt_public_key.pem
```

Store at `infra/secrets/` (directory listed in `.gitignore`). Document the key rotation procedure in `docs/SECRET_ROTATION.md`. The private key becomes a Docker secret in production — never an environment variable string.

### M0.7 — CI skeleton

Write `.github/workflows/ci.yml`:
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `next build` (both web frontends, once they exist)

This runs on every PR and passes immediately since all services are stubs.

**Exit criteria:**
- [ ] Monorepo exists with full directory structure
- [ ] `go work` compiles all service stubs with no errors
- [ ] `docker compose up` starts Postgres + Redis cleanly
- [ ] CI pipeline runs and passes
- [ ] RS256 key pair generated and stored, rotation procedure documented

**Rollback:** Nothing to roll back. M0 is entirely additive and isolated.

---

## Phase M1: Domain Model Unification

**Goal:** Resolve every entity conflict between service-1's schema and the target DOMAIN_MODEL, and write the canonical migration SQL for every table. No code runs against these tables yet — this phase is schema design and migration authoring.

**Entry criteria:** M0 complete.

**Touches live systems:** No.

**What happens:**

### M1.1 — Extract service-1 schema as reference

Open `repos/service-1/api/server.js`. Locate the inline `CREATE TABLE` statements. Copy the schema for each table into a reference file: `docs/service-1-schema-reference.sql`. Do not import it anywhere — this is read-only reference material.

Service-1 tables and their fate:

| service-1 table | Fate | Becomes |
|-----------------|------|---------|
| `users` | Transform | `profiles.users` — remove `points`, keep role, add `global_xp`, `global_level` |
| `venues` | Transform | `catalog.venues` — add `staff_pin`, `theme`, `stripe_account`, `timezone` |
| `events` | Transform | `catalog.events` — rename `date` to `starts_at`, add `ends_at`, `genre`, `image_url` |
| `menu_items` | Transform | `catalog.catalog_items` — add `icon`, `happy_hour_price_nc`, `display_order` |
| `inventory` | Merge | Merged into `catalog.catalog_items` as `stock_qty`, `low_threshold` |
| `orders` | Transform | `orders.orders` — replace `total_points` with `total_nc`, add `idempotency_key`, `ledger_event_id` |
| `order_items` | Keep structure | `orders.order_items` — add name/price snapshots |
| `venue_sessions` | Transform | `sessions.venue_sessions` — add `nitetap_uid`, `ticket_used`, `total_spend_nc`, `checkin_device` |
| `vendor_applications` | Delete | Not NiteOS vocabulary. Market domain only. |
| `vendors` | Delete | Not NiteOS vocabulary. Market domain only. |
| `vendor_products` | Delete | Not NiteOS vocabulary. Market domain only. |
| `quests` | Delete | Deferred to post-pilot gamification layer. |
| `quest_completions` | Delete | Deferred. |
| `notifications` | Delete | Will be re-added in Phase 2 as a new service. |
| `automation_rules` | Delete | Deferred. |
| `logs` | Delete | Replaced by structured JSON logging + Grafana. |
| `analytics_events` | Delete | Replaced by reporting service + ledger events. |

### M1.2 — Write canonical migration SQL

For each table in DOMAIN_MODEL.md, write the migration file. These are the exact files that will run in production. No shortcuts.

Key migration decisions:

**`profiles.users`:**
```sql
-- Remove: points (mutable balance — abolished by Principle 1)
-- Remove: Any vendor-related fields
-- Add: global_xp, global_level (future gamification, defaulting to 0/1)
-- Change: role default stays 'guest' but add constraint
ALTER TABLE profiles.users
  ADD CONSTRAINT users_role_check
  CHECK (role IN ('guest', 'venue_admin', 'bartender', 'door_staff', 'nitecore'));
```

**`catalog.venues`:**
```sql
-- Add: staff_pin (bcrypt hash), theme (jsonb), stripe_account, timezone, slug
-- Not migrated from service-1: service-1 has no production venues to migrate
```

**`ledger.ledger_events`:**
```sql
-- New table. No equivalent in service-1.
-- Critical constraint: no UPDATE or DELETE permissions granted to application roles.
REVOKE UPDATE, DELETE ON ledger.ledger_events FROM niteos_app;
```

### M1.3 — Entity ownership sign-off

Review DOMAIN_MODEL.md against every service listed in SYSTEM_ARCHITECTURE.md. Confirm that:
- Every entity has exactly one owning service
- No entity is owned by two services
- Every cross-service reference is a UUID with no join

Document any ambiguities resolved during this review in DECISION_PRINCIPLES.md under the contradictions table.

**Exit criteria:**
- [ ] All migration SQL files written and reviewed
- [ ] All service-1 tables classified as: transform / delete / merge
- [ ] Zero entities owned by more than one service
- [ ] Migration runs cleanly against local dev Postgres: `make migrate` passes

**Rollback:** Nothing to roll back. M1 is design and SQL files only.

---

## Phase M2: Authentication Unification

**Goal:** Replace the two conflicting auth models (service-1 JWT + service-2 iron-session) with the single auth model defined in AUTH_MODEL.md. Build the auth service and gateway service. Retire service-1's auth. Do not touch service-2.

**Entry criteria:** M0 complete. M1 complete.

**Touches live systems:** service-1 only (prototype, no production users). service-2 is NOT touched.

### M2.1 — Build auth service

Implement in `services/auth/`. Complete functionality:
- `POST /register` — create user via profiles service, issue access + refresh tokens
- `POST /login` — bcrypt compare via profiles service, issue tokens, enforce rate limits
- `POST /refresh` — Redis lookup, issue new access token
- `POST /logout` — revoke tokens in Redis
- `POST /pin` — venue PIN login for staff (validates device token first)
- `GET /jwks` — RS256 public key endpoint

Redis token management:
- `tok:{jti}` — 15-minute TTL access token record
- `ref:{uid}:{hash}` — 30-day refresh token record
- Rate limit keys with sliding window counters

### M2.2 — Build gateway service

Implement in `services/gateway/`. Acts as the authentication choke point:
- Fetch public key from auth `/jwks` on startup, cache it
- On every request: verify RS256 signature, check expiry, check `tok:{jti}` in Redis
- Inject: `X-User-Id`, `X-User-Role`, `X-Venue-Id`, `X-Device-Id`
- Forward to upstream on success; return 401 on any failure
- Special path: `/auth/*` routes directly to auth service (unauthenticated)
- Internal-only header: `X-Internal-Service: {service-name}` — set by gateway for service-to-service calls

### M2.3 — Build profiles service (auth dependency)

Auth service calls profiles service at registration and login time. Build the subset profiles needs:
- `POST /users` — create user (called by auth at registration)
- `GET /users/by-email/{email}` — returns `{user_id, password_hash}` (internal call only, never exposed to clients)
- `GET /users/{user_id}` — fetch user (without password_hash)

These three endpoints are all auth needs from profiles in Phase M2. The rest of profiles is built in M3.

### M2.4 — Deprecate service-1 auth

service-1 uses 7-day non-revocable JWTs signed with a symmetric HS256 key (`JWT_SECRET` environment variable). This is never merged into the new system.

**Actions:**
1. In the NiteOS system, the gateway will reject any service-1 JWT (different signing key, different algorithm, no Redis record)
2. service-1 continues to run on VPS A unchanged — its auth still works for service-1's own frontend
3. No migration of service-1 tokens is attempted — they will expire naturally (7 days) when service-1 is retired in M6

service-1's JWT secret is documented as "superseded" in `docs/ARCHIVE_PLAN.md`. It is never used again.

### M2.5 — Auth system tests

Write integration tests that prove the complete auth flow end-to-end:
- Register → access token in response → token validates through gateway
- Login → token → refresh → new token → original token still valid until expiry
- Logout → refresh token deleted from Redis → refresh attempt returns 401
- Expired access token → 401 from gateway
- Manually deleted `tok:{jti}` from Redis → 401 from gateway
- PIN login with valid device token + correct venue PIN → staff access token issued
- Invalid PIN → 401 (no information about which field was wrong)
- Rate limit: 6th login attempt within 60 seconds from same IP → 429

**Exit criteria:**
- [ ] auth service handles all 6 flows from AUTH_MODEL.md
- [ ] gateway correctly validates and rejects tokens
- [ ] All integration tests pass
- [ ] service-2 is completely unchanged
- [ ] service-1 still runs on VPS A as before

**Rollback:** Remove NiteOS auth + gateway services from the local stack. service-1 was never modified; it continues running.

---

## Phase M3: Service Build and Merge

**Goal:** Build all 13 Go microservices. Work in kernel-first order: ledger before wallet, wallet before orders, orders before reporting. Run the new system in parallel with service-1 on VPS A — two separate stacks, same server, different ports.

**Entry criteria:** M0, M1, M2 complete.

**Touches live systems:** No. The NiteOS stack runs on different ports from service-1.

### M3.1 — Kernel layer (must be sequential, each depends on prior)

**Order:**
1. `ledger` — append-only event store. No external dependencies except Postgres.
2. `profiles` — complete implementation (extends M2.3 stub).
3. `catalog` — venues, catalog items, events. No external dependencies.
4. `wallet` — read-only aggregation over ledger.

**For each service, the build checklist:**
- [ ] All endpoints from DATA_OWNERSHIP.md implemented
- [ ] `GET /healthz` returns 200
- [ ] `GET /metrics` returns Prometheus format
- [ ] All migrations run cleanly
- [ ] Unit tests for domain logic
- [ ] Integration tests against local Postgres

**Ledger service: critical correctness tests**
```
test: write topup_confirmed event → balance = +X NC
test: write order_paid event → balance = +X -Y NC
test: duplicate idempotency_key → returns original event_id, no new row
test: write event with wrong service header → 403
test: attempt UPDATE on ledger_events → permission denied at Postgres level
test: balance projection excludes topup_pending and venue_checkin
```

### M3.2 — Payments layer

5. `payments` — TWINT + Stripe integration

**TWINT integration sequence:**
1. Implement provider interface: `CreateIntent`, `Confirm`, `Capture`, `Refund`, `VerifyWebhook`
2. Implement mock provider for local testing (no TWINT sandbox required yet)
3. Implement Stripe provider against Stripe test mode (test credentials available immediately)
4. Implement TWINT provider against TWINT sandbox (requires Swiss business account credentials)
5. Write webhook receiver with signature verification for both providers
6. Full payment flow test: intent → webhook → ledger event → balance increase

**Do not gate M3 completion on TWINT sandbox credentials.** Build the abstraction with Stripe first. TWINT is added when credentials arrive. The interface never changes.

### M3.3 — Operations layer

**Order matters here due to dependencies:**

6. `devices` — device enrollment, status management, heartbeat
7. `sessions` — depends on profiles (UID resolve) and devices (terminal validation) and ticketing (ticket validation)
8. `ticketing` — depends on payments (confirmation) and ledger (write)
9. `orders` — depends on ledger (balance check + write), catalog (item snapshot), sessions (validate), profiles (UID resolve)

**Orders critical path test:**
```
setup: user with 100 NC balance, open session, catalog with item at 20 NC
test: create order (20 NC) → finalize with NFC tap → order status = paid, balance = 80 NC
test: create order (20 NC) → finalize with only 10 NC balance → 402, order stays pending, balance unchanged
test: void order within 2 min → compensating event, balance restored to 100 NC
test: void order after 2 min by non-admin → 403
```

### M3.4 — Aggregation layer

10. `sync` — edge frame ingestion and cloud write-through
11. `reporting` — read-only aggregation across services

### M3.5 — Edge service

Build `edge/` as a standalone Go binary:
- SQLite with embedded migrations
- Local ledger (identical schema to cloud ledger, restricted to this venue's events)
- LAN API (orders, sessions, catalog cache) on `:9000`
- Catalog sync agent: pull from cloud catalog service on startup, refresh every 15 min
- Sync queue: SQLite table of unsynced events with retry state
- Sync agent: background goroutine, POST sync frames to cloud sync service on connectivity
- NFC reader abstraction
- Cloud fallback: if edge unreachable from terminal, terminals hit cloud directly

**Edge correctness tests:**
```
test: create order offline (cloud unreachable) → local ledger updated, sync queue populated
test: restore connectivity → sync agent flushes queue → cloud ledger contains event
test: replay same sync frame → idempotency_key match → no duplicate, success response
test: edge balance check with only local events → correct
```

### M3.6 — Wire the full stack

Update `infra/docker-compose.dev.yml` to include all 13 services + edge + both Next.js apps (as stubs initially). Confirm all service-to-service calls work in the local stack. Run the full order flow end-to-end:

```
register user → top up via mock payment → receive NC → check in at venue →
bartender opens order → tap NFC → balance checked → order paid → session closed
```

**Exit criteria:**
- [ ] All 13 Go services build and pass their tests
- [ ] Full order flow test passes in local stack
- [ ] Edge service handles offline + sync cycle
- [ ] `service-1` on VPS A is completely unchanged

**Rollback:** NiteOS stack is separate from service-1. Stop the NiteOS Docker stack. service-1 continues running.

---

## Phase M4: Admin Unification

**Goal:** Build the unified admin console (`web/admin/`) to replace service-1's admin Next.js app. Do not touch service-2's admin interface.

**Entry criteria:** M3 complete. All backend services are running in the NiteOS stack.

**Touches live systems:** service-1 admin (prototype only). service-2 admin untouched.

### M4.1 — Identify what service-1 admin does

From `repos/service-1/admin/`, catalogue every page and feature. Classify each:

| service-1 admin feature | NiteOS admin equivalent | Phase |
|-------------------------|-------------------------|-------|
| Venue management (create, edit) | `web/admin/catalog/venues/` | M4 |
| Menu items (CRUD) | `web/admin/catalog/items/` | M4 |
| Event listings (CRUD) | `web/admin/catalog/events/` | M4 |
| User list (view, role change) | `web/admin/profiles/` | M4 |
| Order history | `web/admin/orders/` | M4 |
| Live dashboard (revenue, occupancy) | `web/admin/dashboard/` | M4 |
| Quest management | Not in scope — gamification deferred | Post-pilot |
| Vendor management | Not in NiteOS — Market domain | Never |
| `users.points` direct edit | Abolished — ledger is law | Never |

### M4.2 — Build web/admin/

Build the admin console in Next.js 14 App Router. BFF pattern: all API calls go through Next.js API routes which set httpOnly cookies and forward tokens to backend services.

**CSRF protection (from service-2, reused verbatim):**
Copy `repos/service-2/middleware/csrf.ts` → `web/admin/middleware.ts`. Apply to all mutating routes. This is the pattern: check Origin header matches expected host on all POST/PUT/PATCH/DELETE.

**Image upload (from service-2, reused):**
Copy `repos/service-2/lib/image.ts` (Sharp pipeline, magic-byte validation, 6 output sizes) → `web/admin/lib/image.ts`. Use for venue logo and catalog item images.

**Vendor onboarding pattern (from service-2, adapted):**
service-2's vendor invite token flow maps cleanly to NiteOS venue admin onboarding. Adapt the pattern for `web/admin/onboarding/`.

Build order within admin:
1. Auth (venue_admin login, nitecore login)
2. Catalog management (venues, items, categories, pricing)
3. Device management (enrollment codes, approval, revocation)
4. Live dashboard (tonight's stats from reporting service)
5. Session management (live occupancy, manual close)
6. Refund approval workflow
7. End-of-night report (PDF export)

### M4.3 — Nitecore HQ section

The nitecore role gets a separate section within admin-web (or a separate subdomain, `nitecore.peoplewelike.club`):
- Network health (all venues, sync lag, device heartbeat)
- The Mint (NC in circulation, daily volume, breakage accumulation)
- Venue onboarding (create venue, issue edge token)
- Fraud signals (staff void/refund rate flags from reporting service)
- Revenue view (SaaS billing, transaction fees)

### M4.4 — Parallel operation period

Run both admin systems simultaneously:
- service-1 admin: still accessible, still works against service-1 Postgres
- NiteOS admin: new system, connects to NiteOS services

This is the "shadow period." Venue admins (if any exist) can be onboarded to the new admin while service-1 admin remains a safety net.

**Exit criteria:**
- [ ] All service-1 admin features have NiteOS equivalents built or explicitly deferred
- [ ] Venue admin can create venue, add catalog items, enroll devices
- [ ] Nitecore admin can view network health and mint stats
- [ ] CSRF protection applied to all mutating routes
- [ ] service-2 admin console is completely unchanged

**Rollback:** Stop `web/admin/`. service-1 admin continues working.

---

## Phase M5: Infrastructure Consolidation

**Goal:** Move VPS A from service-1's Docker Compose to the NiteOS Traefik-based Docker Compose. service-1 goes into standby. DNS is not yet switched.

**Entry criteria:** M3, M4 complete. Full NiteOS stack tested in staging.

**Touches live systems:** VPS A. This is the riskiest migration phase because it involves the live server. service-2 and service-3 on VPS B are not touched.

### M5.1 — Prepare VPS A

VPS A currently runs service-1's stack. Before touching it:

1. Take a full VPS snapshot (via VPS provider dashboard)
2. Export service-1's Postgres database: `pg_dump niteos_service1 > service1_backup_$(date +%Y%m%d).sql`
3. Store backup in two places: local and cloud object storage
4. Document service-1's current nginx config and port assignments

service-1's current ports (to avoid conflicts during co-existence):
```
nginx: 80, 443
postgres: 5432 (internal)
api: 3001 (internal)
os: 3000 (internal)
admin: 3002 (internal)
```

### M5.2 — Deploy NiteOS on VPS A (non-conflicting ports)

Deploy the NiteOS stack alongside service-1, using different ports:

```yaml
# infra/docker-compose.cloud.yml (initial VPS A deployment — staging mode)
# Traefik listens on ports 8080 (HTTP) and 8443 (HTTPS) to avoid conflicting with service-1's nginx
```

Confirm the NiteOS stack comes up cleanly on VPS A. Run smoke tests against the staging ports.

### M5.3 — Traefik configuration

Configure Traefik with Cloudflare DNS-01 TLS:
- `traefik.yml` — entrypoints, Cloudflare resolver, dashboard
- `dynamic/routes.yml` — service routing rules

All NiteOS routes go through a staging subdomain for now: `staging.peoplewelike.club` (separate DNS entry, not conflicting with service-1).

Verify:
- TLS certificates issued via Cloudflare DNS-01 for `staging.peoplewelike.club`
- Gateway receives requests and routes to correct services
- Auth flow works through Traefik on staging domain

### M5.4 — Grafana + observability

Deploy Grafana and Prometheus on VPS A. Import dashboards from `infra/grafana/dashboards/`. Confirm:
- All 13 services are visible in Grafana
- Payment failure rate dashboard is populated
- Edge sync health dashboard is ready (shows no venues yet — that's expected)

### M5.5 — Cutover preparation

Before switching DNS:
- Set service-1's nginx to return a maintenance page (do not kill it yet — just prepare the page)
- Ensure NiteOS stack on VPS A is stable under 24 hours of uptime on staging subdomain
- Confirm backup automation runs successfully (`scripts/backup.sh` produces valid backup)
- Confirm at least one full end-to-end test on the staging subdomain: register → top up → order → sync

### M5.6 — VPS A Traefik takeover

Switch VPS A's port 80 and 443 from service-1's nginx to Traefik:

1. Stop service-1's nginx: `docker compose -f /path/to/service-1/docker-compose.yml stop nginx`
2. Reconfigure Traefik to listen on 80 and 443
3. Update Traefik dynamic config to route production domains
4. service-1 API, OS, and Admin containers remain running — they just have no port 80/443 access

DNS is still pointing to service-1 routes at this point — no public traffic sees the change.

**Exit criteria:**
- [ ] VPS A snapshot taken
- [ ] service-1 database backed up
- [ ] NiteOS stack running on VPS A (staging ports)
- [ ] Traefik serving `staging.peoplewelike.club` with valid TLS
- [ ] All smoke tests pass on staging
- [ ] Grafana dashboards populated
- [ ] Backup automation verified

**Rollback:**
1. Stop Traefik
2. Start service-1 nginx
3. service-1 is immediately restored

VPS snapshot is the ultimate fallback: restore VPS to pre-M5 state if something is badly broken.

---

## Phase M6: Cutover and First Pilot

**Goal:** Switch DNS. The first live venue goes on NiteOS. service-1 is parked.

**Entry criteria:** M5 complete. First pilot venue has been provisioned and tested on staging. Android devices enrolled and tested. At least one end-to-end transaction confirmed on staging.

**Touches live systems:** DNS. This is the point of no return for service-1.

### M6.1 — Pilot venue provisioning (on staging)

Before DNS switch, provision the first venue on NiteOS in full:
1. Create venue via Nitecore admin (venue record, staff PIN, catalog)
2. Upload full menu catalog
3. Enroll Master Tablet (configure edge service, provision edge token)
4. Enroll at least one NiteKiosk and one NiteTerminal via Master Tablet
5. Run 20+ test transactions: check-in, order, tap, close session
6. Confirm all events appear in cloud ledger with correct `synced_from`
7. Confirm balance projections are correct after all transactions

### M6.2 — DNS switch

```
os.peoplewelike.club    → VPS A (Traefik → NiteOS guest-web)
admin.peoplewelike.club → VPS A (Traefik → NiteOS admin-web)
api.peoplewelike.club   → VPS A (Traefik → NiteOS gateway)
```

Change in Cloudflare dashboard. TTL should already be set to 60 seconds in advance.

Verify propagation. Confirm TLS certificates are valid on production domains.

### M6.3 — Post-cutover verification

Within the first 30 minutes:
- [ ] `os.peoplewelike.club` loads NiteOS guest web
- [ ] `admin.peoplewelike.club` loads NiteOS admin console
- [ ] Registration and login work
- [ ] TWINT top-up initiated (even if not completed — confirm intent creation reaches TWINT)
- [ ] Check-in on NiteTerminal works
- [ ] Order on NiteKiosk works
- [ ] Sync frame appears in cloud within 30 seconds of order

### M6.4 — service-1 parking

After successful cutover:
1. service-1 containers remain running on VPS A — they just receive no traffic
2. Keep service-1 running for 30 days (3+ successful venue nights)
3. During this period, rollback to service-1 is possible by restoring DNS
4. After 30 days / 3 successful nights: proceed to M7 (service-2 integration) and retire service-1

**Rollback (within 30 days):**
1. Switch DNS back to service-1 routes
2. service-1 containers are still running; no data migration needed
3. Any transactions that occurred on NiteOS during the window are in the NiteOS ledger (cannot be backported to service-1, but service-1 had no production transactions before cutover anyway)

**Exit criteria:**
- [ ] DNS switched and propagated
- [ ] First pilot venue live on NiteOS
- [ ] 3+ successful venue nights completed
- [ ] No transaction errors in Grafana
- [ ] service-1 parked but not yet retired

---

## Phase M7: service-2 Auth Integration

**Goal:** Add shared identity between NiteOS and People We Like Market. Additive only — service-2 continues working with its existing iron-session auth throughout.

**Entry criteria:** M6 complete. NiteOS running stably for 30+ days. NiteOS auth service has a stable API.

**Touches live systems:** service-2 (Market). This is additive. Existing auth path is never removed.

### M7.1 — NiteOS auth adds OAuth2/OIDC server

Add OAuth2 authorization server endpoints to `services/auth/`:
- `GET /oauth/authorize` — authorization endpoint
- `POST /oauth/token` — token endpoint (authorization code exchange)
- `GET /oauth/userinfo` — userinfo endpoint (returns uid, email, display_name)

Register service-2 as a client: `client_id`, `client_secret`, allowed redirect URI.

### M7.2 — service-2 adds "Sign in with NiteOS" (additive)

In `repos/service-2/`, add a new login option alongside existing iron-session auth:

```typescript
// New: /app/auth/niteos-callback/route.ts
// Exchange OAuth code for NiteOS user info
// Link or create service-2 account with niteos_user_id
```

Add `niteos_user_id` nullable column to service-2's users table. Existing users are unaffected.

### M7.3 — Shared balance display (optional, Phase 2)

Once shared identity is established, NC wallet balance can be surfaced in service-2:
- service-2 frontend calls NiteOS wallet API using the user's linked NiteOS access token
- Balance is read-only in service-2 — no payments are initiated from Market

This is out of scope for M7 itself. M7 delivers shared identity only. Balance display is Phase 2 scope.

**Exit criteria:**
- [ ] service-2 users can log in via NiteOS OAuth
- [ ] service-2 iron-session auth still works unchanged
- [ ] NiteOS users see Market linked in their profile
- [ ] No service-2 data was migrated or deleted

---

## Migration Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| TWINT sandbox credentials delayed | High | Payments blocked | Build with Stripe first; TWINT is a drop-in swap |
| VPS A instability during M5 | Medium | service-1 downtime | VPS snapshot taken first; Traefik staged on non-conflicting ports |
| Edge LAN reliability in pilot venue | Medium | Bartenders hit cloud | Cloud fallback is built-in; edge failure degrades to cloud, doesn't fail |
| service-2 integration breaks Market auth | Low | Market outage | Iron-session auth path never removed; OAuth is additive |
| Android device enrollment in real venue | Medium | Terminal can't go live | Dry-run enrollment 48h before first live night |
| Sync conflict (duplicate event IDs) | Low | Incorrect ledger | Idempotency key unique constraint; tested in M3 with replay tests |
| Ledger balance race condition | Low | Overspend | Balance check + write-time enforcement in ledger service; tested in M3 |

---

## What Is Never Migrated

| Item | Reason |
|------|--------|
| service-1 `users.points` values | Mutable balance is abolished. No equivalent concept in NiteOS. |
| service-1 JWT tokens | Different algorithm, different key, no Redis record. Will expire in 7 days. |
| service-1 quest/XP data | Gamification deferred to post-pilot. |
| service-1 vendor tables | Market domain. Not NiteOS vocabulary. |
| service-2 iron-session cookies | service-2 stays on iron-session. Its auth is not migrated; it gains a second login option. |
| service-2 taste graph data | Not NiteOS. Stays in service-2. |
| service-3 configuration | Radio is not migrated. It stays as-is forever. |
