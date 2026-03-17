# FINAL_REPO_STRUCTURE.md

The definitive repository layout for the NiteOS system. This is the target state. Current state is three disconnected repos; this is what replaces them.

---

## Repository Strategy

**One monorepo, one product.**

All NiteOS services, frontends, and tooling live in a single repository. The radio service (service-3) stays in its own repo — it is a separate product on a separate VPS and has no coupling to NiteOS. The market service (service-2) stays separate until the shared identity layer is built in Phase 2.

This is not a monolith. It is a monorepo of microservices. Each service is independently deployable. The monorepo exists because:
- Atomic commits across services (schema change + consumer change in one PR)
- Shared tooling (linters, proto definitions, CI configuration)
- Single source of truth for service contracts
- No version drift between services during development

---

## Root Structure

```
niteos/
├── .github/
│   └── workflows/
│       ├── ci.yml                  # build + test all services on PR
│       ├── deploy-cloud.yml        # deploy to NiteOS VPS (31.97.126.86) on merge to main
│       └── deploy-edge.yml         # build edge binary for distribution
│
├── services/                       # Go microservices
│   ├── gateway/
│   ├── auth/
│   ├── profiles/
│   ├── ledger/
│   ├── wallet/
│   ├── payments/
│   ├── ticketing/
│   ├── orders/
│   ├── catalog/
│   ├── devices/
│   ├── sessions/
│   ├── reporting/
│   └── sync/
│
├── edge/                           # Edge node (Go, SQLite, runs on Master Tablet)
│
├── web/                            # Next.js frontends
│   ├── guest/                      # os.peoplewelike.club
│   └── admin/                      # admin.peoplewelike.club
│
├── android/                        # Kotlin Android apps
│   ├── nitekiosk/
│   ├── niteterminal/
│   └── mastertablet/
│
├── infra/                          # Deployment configuration
│   ├── docker-compose.cloud.yml    # NiteOS VPS production stack (31.97.126.86)
│   ├── docker-compose.dev.yml      # Local development stack
│   ├── traefik/
│   │   ├── traefik.yml
│   │   └── dynamic/
│   │       └── routes.yml
│   └── grafana/
│       ├── dashboards/
│       └── provisioning/
│
├── migrations/                     # Postgres schema migrations (per service)
│   ├── profiles/
│   ├── ledger/
│   ├── payments/
│   ├── ticketing/
│   ├── orders/
│   ├── catalog/
│   ├── devices/
│   ├── sessions/
│   └── sync/
│
├── docs/                           # Architecture documentation (this directory)
│
├── scripts/
│   ├── dev-up.sh                   # Start local development stack
│   ├── dev-down.sh
│   ├── seed.sh                     # Seed database with test venue + user
│   ├── backup.sh                   # (from service-2 systemd backup pattern)
│   └── migrate.sh                  # Run pending migrations for all services
│
├── go.work                         # Go workspace file (links all service modules)
├── go.work.sum
├── Makefile                        # Top-level build targets
└── README.md
```

---

## Service Structure (repeated for each Go service)

Each service under `services/` follows the same layout. Example: `services/orders/`

```
services/orders/
├── cmd/
│   └── main.go                     # Entry point: reads config, wires deps, starts server
│
├── internal/
│   ├── config/
│   │   └── config.go               # Env var loading (no config files — 12-factor)
│   │
│   ├── handler/
│   │   ├── orders.go               # HTTP handlers (thin: parse → call domain → respond)
│   │   └── health.go               # GET /healthz
│   │
│   ├── domain/
│   │   ├── order.go                # Order entity, business rules, domain errors
│   │   └── finalize.go             # Order finalization logic (balance check, ledger write)
│   │
│   ├── store/
│   │   └── postgres.go             # SQL queries (no ORM — plain pgx)
│   │
│   └── client/
│       ├── ledger.go               # HTTP client for ledger service
│       ├── profiles.go             # HTTP client for profiles service
│       ├── sessions.go             # HTTP client for sessions service
│       └── catalog.go              # HTTP client for catalog service
│
├── go.mod
├── Dockerfile
└── README.md                       # Service-specific: what it does, env vars, API endpoints
```

**No global state.** No init() functions with side effects. No package-level database connections. Everything injected via constructor.

**No ORMs.** Plain `pgx/v5` for all Postgres queries. SQL lives in the `store` package, not scattered through handlers.

**No shared internal packages between services.** If two services need the same helper, it is either copied (preferred for small things) or extracted to a published Go module (for substantial shared logic like JWT parsing).

---

## Shared Go Module

```
niteos/
└── pkg/                            # Shared Go code (published as a module within the workspace)
    ├── jwtutil/
    │   └── jwtutil.go              # JWT parsing and validation (RS256). Used by: gateway, auth
    ├── middleware/
    │   └── auth.go                 # HTTP middleware that trusts X-User-Id headers. Used by: all services
    ├── httputil/
    │   └── respond.go              # JSON response helpers. Used by: all services
    └── idempotency/
        └── key.go                  # Idempotency key generation convention. Used by: orders, payments, sessions, ticketing
```

This package is intentionally minimal. Services depend on it — it must not depend on any service. It contains no business logic.

---

## Frontend Structure

```
web/guest/                          # Guest web app (os/www/apex.peoplewelike.club)
├── app/                            # Next.js 14 App Router
│   ├── layout.tsx                  # Root layout — dark theme, mobile-first
│   ├── page.tsx                    # Home: landing (logged out) / event feed + wallet (logged in)
│   ├── globals.css
│   ├── (auth)/
│   │   ├── login/page.tsx
│   │   └── register/page.tsx
│   ├── wallet/
│   │   ├── page.tsx                # Balance display + top-up
│   │   └── TopUpButton.tsx         # Client component — initiates Stripe checkout
│   ├── tickets/page.tsx            # My tickets (active + past)
│   ├── events/page.tsx             # Venue/event discovery feed
│   ├── session/page.tsx            # Active venue session (post NiteTap check-in)
│   ├── venues/[slug]/page.tsx      # Venue detail + catalog preview
│   └── api/                        # BFF routes (httpOnly cookies — no direct backend calls from browser)
│       ├── auth/
│       │   ├── login/route.ts
│       │   ├── register/route.ts
│       │   └── logout/route.ts
│       ├── wallet/topup/route.ts   # Initiates Stripe checkout → returns checkout_url
│       └── health/route.ts
├── components/
│   ├── NavBar.tsx                  # Sticky nav with wallet badge, mobile bottom nav
│   ├── LogoutButton.tsx            # Client component
│   └── WalletBadge.tsx             # Wallet summary card
├── lib/
│   ├── api.ts                      # Server-side BFF helpers (backendFetch, typed API calls)
│   └── session.ts                  # iron-session (24h, httpOnly, GuestSession type)
├── .env.local.example
├── next.config.ts
├── package.json
├── tailwind.config.ts
└── Dockerfile

web/admin/                          # Admin console (admin.peoplewelike.club)
├── app/
│   ├── (auth)/
│   ├── dashboard/
│   ├── catalog/
│   ├── devices/
│   ├── sessions/
│   ├── reports/
│   ├── refunds/
│   └── api/                        # BFF routes
├── components/
├── lib/
├── package.json
└── Dockerfile
```

**No direct API calls from browser JavaScript.** All backend calls go through Next.js API routes (BFF pattern). This is where JWTs are stored in httpOnly cookies and forwarded as Authorization headers to backend services.

---

## Edge Structure

```
edge/
├── cmd/
│   └── main.go                     # Entry point: starts LAN API, sync agent, NFC daemon
│
├── internal/
│   ├── config/
│   │   └── config.go               # Reads /etc/niteos-edge/config.toml
│   │
│   ├── db/
│   │   ├── sqlite.go               # SQLite connection + WAL mode setup
│   │   └── migrations/             # SQLite schema migrations (embedded in binary)
│   │
│   ├── api/
│   │   ├── orders.go               # LAN API: order creation and finalization
│   │   ├── sessions.go             # LAN API: check-in and check-out
│   │   ├── catalog.go              # LAN API: serve local catalog cache
│   │   └── health.go
│   │
│   ├── ledger/
│   │   └── local.go                # Local ledger: write events, compute balance projection
│   │
│   ├── sync/
│   │   ├── queue.go                # Sync queue: buffer unsent events
│   │   ├── frame.go                # Assemble SyncFrame (checksum, event range)
│   │   └── agent.go                # Background goroutine: flush queue to cloud sync service
│   │
│   ├── catalog/
│   │   └── cache.go                # Pull catalog from cloud, serve locally
│   │
│   └── nfc/
│       └── reader.go               # NFC reader interface (USB HID abstraction)
│
├── go.mod
├── Dockerfile                      # For Master Tablet deployment
└── README.md                       # Setup guide: config file format, required env, hardware setup
```

---

## Android Structure

```
android/
├── nitekiosk/                      # Bartender POS terminal
│   ├── app/
│   │   ├── src/main/
│   │   │   ├── java/com/niteos/kiosk/
│   │   │   │   ├── ui/             # Jetpack Compose screens
│   │   │   │   │   ├── catalog/    # Item selection grid
│   │   │   │   │   ├── order/      # Active order + total
│   │   │   │   │   └── confirm/    # Tap to pay screen
│   │   │   │   ├── nfc/            # NFC foreground dispatch
│   │   │   │   ├── network/        # API client (LAN-first, cloud fallback)
│   │   │   │   ├── auth/           # Device token storage (Android KeyStore)
│   │   │   │   └── kiosk/          # Device Owner policy enforcement
│   │   │   └── AndroidManifest.xml
│   │   └── build.gradle.kts
│   └── settings.gradle.kts
│
├── niteterminal/                   # Door staff check-in terminal
│   ├── app/
│   │   ├── src/main/
│   │   │   ├── java/com/niteos/terminal/
│   │   │   │   ├── ui/
│   │   │   │   │   ├── scan/       # QR scanner screen
│   │   │   │   │   ├── nfc/        # NFC tap screen
│   │   │   │   │   └── status/     # Check-in result display
│   │   │   │   ├── camera/         # QR code scanning (CameraX + ML Kit)
│   │   │   │   ├── nfc/
│   │   │   │   ├── network/
│   │   │   │   ├── auth/
│   │   │   │   └── kiosk/
│   │   │   └── AndroidManifest.xml
│   │   └── build.gradle.kts
│   └── settings.gradle.kts
│
├── mastertablet/                   # Edge node host + venue admin UI
│   ├── app/
│   │   ├── src/main/
│   │   │   ├── java/com/niteos/master/
│   │   │   │   ├── ui/
│   │   │   │   │   ├── dashboard/  # Live tonight view
│   │   │   │   │   ├── devices/    # Device enrollment
│   │   │   │   │   ├── sync/       # Sync status
│   │   │   │   │   └── endofnight/ # Close + export
│   │   │   │   ├── edge/           # Embedded edge service manager
│   │   │   │   ├── network/
│   │   │   │   └── auth/
│   │   │   └── AndroidManifest.xml
│   │   └── build.gradle.kts
│   └── settings.gradle.kts
│
└── shared/                         # Kotlin multiplatform module (shared by all 3 apps)
    └── src/
        ├── network/
        │   ├── EdgeApiClient.kt    # LAN API calls
        │   └── CloudApiClient.kt   # Cloud fallback calls
        ├── auth/
        │   └── KeyStoreManager.kt  # Device token storage in Android KeyStore
        └── model/
            └── Models.kt           # Shared data classes
```

---

## Infrastructure

```
infra/
├── docker-compose.cloud.yml        # Production: all 13 Go services + 2 Next.js apps + Traefik
├── docker-compose.dev.yml          # Dev: same stack + Postgres + Redis exposed on localhost
│
├── traefik/
│   ├── traefik.yml                 # Entrypoints (80, 443), Cloudflare DNS-01 resolver, dashboard
│   └── dynamic/
│       └── routes.yml              # Service routing rules, middleware chains
│
└── grafana/
    ├── provisioning/
    │   ├── datasources/
    │   │   └── prometheus.yml
    │   └── dashboards/
    │       └── dashboards.yml
    └── dashboards/
        ├── service-health.json     # All 13 services: uptime, latency, error rate
        ├── payments.json           # Payment failure rate, webhook lag, reconciliation delta
        ├── edge-sync.json          # Per-venue: unsynced events, sync age, edge node status
        └── devices.json            # Per-venue: device heartbeat, last seen, firmware version
```

---

## Migrations

```
migrations/
├── profiles/
│   ├── 001_create_users.sql
│   ├── 002_create_venue_profiles.sql
│   └── 003_create_nitetaps.sql
│
├── ledger/
│   ├── 001_create_ledger_events.sql
│   └── 002_add_synced_from_index.sql
│
├── catalog/
│   ├── 001_create_venues.sql
│   ├── 002_create_catalog_items.sql
│   ├── 003_create_happy_hour_rules.sql
│   └── 004_create_events.sql
│
├── [... one directory per service with a Postgres schema ...]
```

Migrations run via `scripts/migrate.sh` (pure psql — no golang-migrate dependency). The script applies all SQL files in order using a psql connection. The CI pipeline runs a migrations dry-run against a test database before any deployment. No golang-migrate binary is required.

---

## What Survives From Existing Repos

### From service-1 (repos/service-1/)

| Component | Fate | Notes |
|-----------|------|-------|
| `api/server.js` — venue, user, event schema | Reference only | DB schema ported to Go services with domain corrections (no mutable points, no vendor tables) |
| `api/server.js` — PIN auth concept | Pattern reused, implementation rewritten | PIN hashed with bcrypt, validated by auth service not inline |
| `api/server.js` — inventory/menu model | Ported to catalog service | Restructured as CatalogItem entity |
| `api/server.js` — guest check-in concept | Ported to sessions service | Rewritten with NiteTap UID model |
| `api/server.js` — quest/XP system | Deferred | Not ported to MVP |
| `api/server.js` — vendor/vendor_products | Deleted | Belongs to service-2 domain. Not NiteOS vocabulary. |
| `os/` — Next.js frontend | Reference only | Guest web rebuilt in `web/guest/` using App Router |
| `admin/` — Next.js admin | Reference only | Admin console rebuilt in `web/admin/` |
| `docker-compose.yml` — compose pattern | Pattern evolved | Infra compose files extend this pattern with Traefik |

**Nothing from service-1 is directly copied.** The codebase is rewritten in Go (backend) and TypeScript/Next.js App Router (frontend). service-1 is reference material only.

### From service-2 (repos/service-2/)

| Component | Fate | Notes |
|-----------|------|-------|
| `lib/image.ts` — Sharp image pipeline | Reused as-is | Copied into `web/admin/lib/image.ts` for product image uploads |
| `lib/session.ts` — iron-session pattern | Pattern reused, implementation changed | Replaced with JWT httpOnly cookie pattern in Next.js BFF (no iron-session) |
| `middleware/csrf.ts` — Origin header CSRF | Reused verbatim | Copied into both `web/guest` and `web/admin` middleware |
| `lib/metrics.ts` — Prometheus endpoint pattern | Reused | Go services implement equivalent `GET /metrics` endpoint using `prometheus/client_golang` |
| `scripts/backup.sh` — systemd backup timer | Reused in `scripts/backup.sh` | Adapted for NiteOS data directories |
| `app/admin/vendors/` — vendor onboarding flow | Pattern reused | Adapted for venue admin onboarding flow in `web/admin` |
| `db/schema.ts` — Drizzle schema patterns | Reference | Informs domain model design; not ported (backend is Go with pgx, not Drizzle) |
| `db/schema.ts` — taste graph entities | Not ported | Taste graph is service-2 specific (market feature) |
| Entire market product | Unchanged | service-2 remains its own repo and deployment |

### From service-3 (repos/service-3/)

| Component | Fate | Notes |
|-----------|------|-------|
| Entire radio stack | Unchanged | Stays in its own repo. service-3 is complete and working. No changes. |
| RTMP ingest, AutoDJ, switch daemon, HLS relay | Unchanged | Not part of NiteOS |
| `radio-web/` player | Unchanged | Embedded in guest-web via iframe/embed only |

---

## CI/CD Pipeline

### On Pull Request (`ci.yml`)

```
1. go build ./...           — build all services
2. go vet ./...             — static analysis
3. go test ./...            — run all unit tests
4. migrations dry-run       — verify SQL parses and applies cleanly to test DB
5. docker build             — build all service images (no push)
6. next build               — build both web frontends
```

### On Merge to Main (`deploy-cloud.yml`)

```
1. Run CI checks (above)
2. docker build + push all images to registry
3. SSH to NiteOS VPS (31.97.126.86)
4. docker compose pull       — pull new images
5. docker compose up -d      — rolling restart (Compose recreates changed services)
6. Run migrations            — pending migrations apply automatically on service startup
7. Smoke test                — curl /healthz on all services
8. Alert on failure          — notify via configured channel (Phase 1: email; Phase 2: Slack/webhook)
```

### On Edge Release (`deploy-edge.yml`)

```
1. go build -o edge-node ./edge/cmd/     — produce static Linux ARM binary
2. Upload to GitHub Releases             — tagged release artifact
3. Master Tablet app downloads on next update check
```

---

## Environment Variables (per service, standardized names)

All services read configuration from environment variables only (12-factor). No config files in application code. No hardcoded values.

```
# Database (all services with Postgres)
DATABASE_URL=postgres://niteos:xxx@db:5432/niteos?sslmode=require

# Redis (auth, gateway, rate-limiting services)
REDIS_URL=redis://:xxx@redis:6379

# JWT (auth service only — holds private key)
JWT_PRIVATE_KEY_FILE=/run/secrets/jwt_private_key
JWT_PUBLIC_KEY_FILE=/run/secrets/jwt_public_key

# Service discovery (each service knows only what it calls)
PROFILES_SERVICE_URL=http://profiles:8020
LEDGER_SERVICE_URL=http://ledger:8030
# ... etc

# Payment providers (payments service only)
TWINT_API_KEY_FILE=/run/secrets/twint_api_key
STRIPE_SECRET_KEY_FILE=/run/secrets/stripe_secret_key
STRIPE_WEBHOOK_SECRET_FILE=/run/secrets/stripe_webhook_secret

# Edge (edge service only)
EDGE_TOKEN_FILE=/etc/niteos-edge/config.toml
CLOUD_SYNC_URL=https://api.peoplewelike.club/sync

# Observability
LOG_LEVEL=info                  # debug | info | warn | error
SERVICE_NAME=orders             # injected by compose, used in structured logs
```

Secrets (keys, passwords, API tokens) are passed as Docker secrets (files), never as environment variable string values in production.

---

## What Is NOT In This Repo

| Item | Location | Why separate |
|------|----------|-------------|
| Radio stack | `repos/service-3/` (own repo) | Separate product, separate VPS, no coupling |
| Market (People We Like Market) | `repos/service-2/` (own repo) | Separate product, separate domain, Phase 2 integration |
| NiteTap hardware design files | TBD (hardware partner) | Physical design files, not software |
| SSL certificates | Traefik + Cloudflare DNS-01 (auto-managed) | Never committed to git |
| Docker secrets / `.env` production | NiteOS VPS filesystem only | Never committed to git |
| Grafana data | NiteOS VPS `/var/lib/grafana` volume | Runtime state, backed up separately |
| Postgres data | NiteOS VPS `/var/lib/postgres` volume | Runtime state, backed up via `scripts/backup.sh` |
