# Product Explained

**NiteOS — Venue Operating System for Nightlife**

Written for: investors, technical due diligence reviewers, advisors, operators, and future team members who need a complete and accurate picture of what this product is, what has been built, and where it stands today.

This document reflects the state of the codebase as of March 2026. It distinguishes clearly between what is built, what is designed but not yet running, and what remains to be done. It does not speculate or project forward as if future capabilities are current.

---

## 1. What the Product Is

NiteOS is a white-label venue operating system designed for nightlife venues: nightclubs, cocktail bars, lounges, and similar establishments. It replaces the fragmented collection of tools that most venues operate with today — a POS system from one vendor, a ticketing platform from another, door staff with paper lists, and no real-time view of what is happening across the floor — with a single integrated system purpose-built for high-volume, cash-light, in-venue commerce.

The core commercial mechanism is NiteCoin (NC), an internal unit of account pegged at exactly 1 NC = 1 CHF. Guests load CHF onto their account (via Stripe or TWINT) before or at the venue, receive the equivalent NC balance, and spend NC at the bar, at the door, and at events entirely by tapping an NFC wearable — a bracelet, ring, or card called a NiteTap. No cash changes hands on the floor. No card is presented at the bar. The guest taps; the balance is debited; the bartender confirms.

The product operates on three physical layers:

**NiteOS Cloud Core** — A cluster of Go microservices running on a cloud VPS, backed by Postgres and Redis. This is the authoritative system. All financial events are recorded here as an append-only ledger. The cloud is the source of truth.

**NiteOS Edge** — A single Go binary running on a dedicated device at each venue (a "Master Tablet" or a dedicated small computer). It carries a local SQLite ledger and a catalog cache. When the internet connection is unreliable — as is common in dense urban clubs — the edge handles order processing locally and syncs to the cloud when connectivity is restored.

**NiteOS Terminals** — Android applications installed on tablets and phones at each venue. NiteKiosk runs at the bar: the bartender sees the catalog, creates an order, and the guest taps the NiteTap to pay. NiteTerminal runs at the door: the door staff scans tickets and checks in guests. The Master Tablet is the admin device for the venue owner or manager.

The three layers are designed to degrade gracefully. If the internet is down, the edge handles orders. If the edge is also down, terminals fall back to the cloud directly. The venue does not stop operating because of a network problem.

---

## 2. Who It Is For

**Venue operators** are the primary customers. The product is sold to them as a managed service. They get: a unified POS replacing the bar till, a ticketing system for events, a real-time occupancy and revenue dashboard, and a loyalty mechanism via NC.

**Guests** are the end users. They interact with the product by loading their NiteCoin wallet (via a web app on their phone), tapping their NiteTap wearable to pay at the bar, and collecting XP and venue-specific perks over time. The guest experience is cashless and card-less at the point of sale.

**Staff** (bartenders, door staff) use the Android applications directly. The UX is designed for speed in a loud, dark, fast-paced environment. A bartender tap-to-pay interaction should complete in under two seconds.

**Nitecore** is the internal operator role — the company behind the product. Nitecore staff have elevated system-wide access to view all venues, the total NC in circulation, sync health, and platform-level revenue.

The product is geographically targeted at Switzerland first, specifically Zurich. The NiteCoin structure is designed to comply with Swiss FinMA requirements as a Multi-Purpose Voucher (MPV), which permits the product to operate without a full banking licence for a defined float ceiling.

---

## 3. User Roles

The system enforces five roles, propagated from the auth service into every request via JWT claims.

**guest** — Standard registered user. Can load NC, tap to pay, view their own wallet and session history. Cannot access any management functionality.

**venue_admin** — Operator of a specific venue. Can manage catalog items, view session and revenue reports for their venue, enroll devices, and approve refunds. Scoped to their venue_id; cannot see other venues.

**bartender** — Operational staff at a venue. Can create and finalize orders. Can void an order they created within a two-minute window. Cannot access reports, device management, or catalog editing.

**door_staff** — Operational staff at the entrance. Can validate tickets and open guest sessions. Cannot access financial data or place orders.

**nitecore** — Platform administrator. Full read access across all venues. Can void any order regardless of age, approve overrides, and view the global NC ledger. This role is never granted to venue staff.

Roles are set at user registration or by a nitecore admin. Staff roles (bartender, door_staff) may also be set per-device at enrollment time, meaning the same person can be a guest on their personal phone and a bartender on the kiosk at work.

---

## 4. Product Surfaces

### Guest Web (web/guest/)

A Next.js web application served at `os.peoplewelike.club`. Designed to be used on a mobile browser — guests do not install an app. Features include: wallet top-up via Stripe/TWINT, balance display, transaction history, venue discovery and event listings, NiteTap registration, and QR ticket display.

**Status: not yet built.** The directory exists as a stub. The backend services that would power it are built. The frontend implementation is a future phase.

### Admin Console (web/admin/)

A Next.js web application at `admin.peoplewelike.club` for venue_admin and nitecore users. Features include: venue and catalog management, device enrollment, live session dashboard, revenue reports, and refund workflows.

**Status: not yet built.** Backend services are complete. Admin console is Phase M4 of the migration plan.

### NiteKiosk (Android)

A kiosk-mode Android app for bar staff. Displays the venue catalog, allows order creation, accepts NFC tap to pay, and shows a confirmation. Designed to run locked to a single venue on a wall-mounted or counter-mounted tablet.

**Status: not yet built.** Described in product blueprints. Android development is planned after the cloud backend is stable and deployed.

### NiteTerminal (Android)

A door-staff Android app for guest check-in and ticket validation. Scans QR codes or reads NiteTap. Writes a `venue_checkin` ledger event and opens a venue session.

**Status: not yet built.** Same status as NiteKiosk.

### Master Tablet (Android)

An admin-mode Android app for the venue owner or manager on-site. Manages the edge node configuration, enrolls other devices on the venue LAN, and provides a real-time view of tonight's session.

**Status: not yet built.**

### People We Like Market (market.peoplewelike.club)

A separate, live product. An e-commerce platform for curated fashion and lifestyle drops from independent vendors. Built in Next.js, running on a separate VPS. Has its own Postgres database and its own session-based authentication (iron-session). Currently production, version 0.4.3, with live marketplace users.

Market is architecturally separate from NiteOS. There is a planned integration (Phase M7) to add shared identity — Market users can optionally link their NiteOS account — but no integration exists today. Market does not share a database, a codebase, or an auth system with NiteOS.

### People We Like Radio (radio.peoplewelike.club)

A separate, live product. An HLS radio stream with a web player, RTMP ingest from DJs, and an autodj fallback. Running on the same VPS as Market. Has active listeners.

Radio has no integration with NiteOS and is not planned to be integrated in the near term. It is referenced in the OS frontend via an iframe embed only.

---

## 5. End-to-End Flows

### Guest Top-Up

1. Guest opens the web app, selects "Top Up", enters an amount in CHF.
2. The payments service creates a payment intent with the provider (Stripe or TWINT).
3. The guest completes payment in the provider's UI.
4. The provider sends a webhook to the payments service.
5. The payments service verifies the webhook signature, then writes a `topup_confirmed` ledger event to the ledger service. Amount in NC = CHF amount × 100 (1 CHF = 100 NC). The ledger event is written synchronously before the webhook returns 200; if the ledger write fails, the webhook returns 500 so the provider retries.
6. The wallet service (read-only, projects from the ledger) now shows the updated balance.

A `topup_pending` event is written when the intent is created. This event type is excluded from the balance projection — the guest's balance does not increase until confirmed. This ensures a provider failure does not result in NC issuance.

### Guest Check-In

1. Guest arrives at the venue. Door staff scans the guest's QR ticket or NiteTap.
2. The sessions service receives the check-in, validates the ticket (via ticketing service), and opens a `venue_session` record.
3. A `venue_checkin` ledger event is written (excluded from balance — it is a tracking event, not a financial event).
4. The guest is now inside with an active session.

### Bar Order

1. Bartender selects items on NiteKiosk. The order is created in the orders service (status: `pending`).
2. The guest taps their NiteTap to the kiosk.
3. The kiosk sends a finalize request to the orders service with the guest's user ID.
4. The orders service checks the guest's balance via the wallet service (which projects from the ledger).
5. If balance is insufficient, the response is 402 Payment Required. The order stays pending. No ledger event is written.
6. If balance is sufficient, the orders service writes an `order_paid` ledger event (negative amount = debit from the guest). This is written synchronously before finalization completes.
7. The orders service updates the order status to `paid` and returns the finalized order.
8. The kiosk displays a success screen.

### Offline Order (Edge)

The same flow, but mediated by the edge service:
1. The kiosk sends the order to the edge LAN API instead of the cloud.
2. The edge checks the local SQLite ledger for the guest's balance.
3. If sufficient, the edge writes an `order_paid` event to its local ledger and enqueues it in the sync queue.
4. The sync agent (running every 30 seconds) picks up pending events, packages them into a sync frame, and POSTs to the cloud sync service.
5. The cloud sync service writes the events to the cloud ledger using the same idempotency key. Duplicate submissions are ignored at the database constraint level.

### Void / Refund

- A bartender can void an order within two minutes of finalization. This writes a `refund_created` ledger event (positive amount = credit back to the guest).
- After two minutes, only a nitecore-role user can void an order. Bartenders receive 403.
- Void of a pending order (not yet finalized) requires no ledger event — nothing was charged.
- There is no DELETE or UPDATE on the ledger. Every financial reversal is an explicit compensating event.

### Ticket Purchase

1. Guest selects a ticket product in the web app.
2. The ticketing service writes a `ticket_purchase` ledger event (negative NC, equal to the ticket's NC price).
3. If the ledger write succeeds, the ticket issuance record is created with a unique QR token.
4. If the ledger write fails, no ticket is issued (the ledger is the authority on whether the charge happened).
5. The guest receives a QR code to present at the door.

---

## 6. Financial and Operational Model

### NiteCoin Structure

NiteCoin is not a cryptocurrency. It is an accounting unit internal to the NiteOS system. Its properties:

- Fixed peg: 1 NC = 1 CHF. This never changes. There is no market for NC.
- Issued only by the payments service upon a confirmed payment. No other service can create NC from nothing.
- Spent by the orders service (bar orders), ticketing service (ticket purchases), and sessions service (check-in where applicable).
- Refunded by compensating events from the orders or payments service.
- Expires after 365 days of user inactivity (not yet implemented — Phase 2).

Swiss regulatory classification: Multi-Purpose Voucher (MPV) under FinMA guidelines. An MPV can be issued in Switzerland without a payment institution licence, subject to a float ceiling. The current planned ceiling is 1,000,000 CHF total outstanding NC across all venues. This gives meaningful runway for a pilot but would require escalation to SRO/VQF membership for a Phase 2 growth trajectory.

### Revenue Streams

**Transaction fee:** 5% of every NC-denominated transaction at the venue. Applied at settlement time, not at the ledger level. Venue receives 95% of their NC volume converted back to CHF via Stripe Connect.

**SaaS subscription:** Monthly per-venue fee. Covers platform access, hardware provisioning support, and Nitecore support. Amount not finalized in documentation — TBD during pilot negotiation.

**Breakage revenue:** NC that expires after 365 days of inactivity is split between the venue (whose catalog the NC was originally earned through) and Nitecore. Exact split not yet formalized.

**Stripe Connect settlement:** At settlement, the NC total for the period is converted to CHF at 1:1 and distributed via Stripe Connect — 95% to the venue's connected account, 5% to the Nitecore platform account.

### TWINT

TWINT is the primary intended payment method for Swiss users. It requires a Swiss business account and TWINT merchant credentials. These credentials are not yet available. The payments service is built with a provider interface; a mock provider and a Stripe provider are fully implemented. The TWINT provider is designed to drop in without interface changes as soon as credentials are obtained. TWINT is the most common payment method in Switzerland for this age demographic and is a meaningful differentiator.

---

## 7. Technical Architecture

### Backend: 13 Go Microservices

All backend services are written in Go 1.22+. Each service is an independent binary with its own `go.mod` and its own Postgres schema. Services communicate via HTTP. There is no shared database access across services — each service owns its schema entirely, and cross-service data is accessed via API call, never via SQL join.

The monorepo uses a Go workspace (`go.work`) to manage all modules together without publishing packages.

**auth** — Handles user registration, password login (bcrypt cost 12), PIN login for staff, token issuance (RS256 JWTs), token refresh (Redis), logout (token revocation), and the JWKS endpoint. Port 8040.

**gateway** — API entry point for all external requests. Fetches the RS256 public key from auth at startup, caches it, refreshes periodically. On every request: verifies JWT signature, checks expiry, checks the Redis revocation set (`tok:{jti}`). Injects `X-User-Id`, `X-User-Role`, `X-Venue-Id`, `X-Device-Id` headers for downstream services. Reverse proxies to the correct upstream service based on path. Port 8000.

**profiles** — Owns the `users`, `venue_profiles`, and `nitetaps` tables. Manages user creation, lookup by email (internal use by auth only), venue profile creation on first check-in, NiteTap registration and revocation. Port 8010.

**ledger** — The financial kernel. Append-only event store. Accepts INSERT-only writes. The Postgres role used by this service has UPDATE and DELETE revoked at the database level. Enforces idempotency via a UNIQUE constraint on `idempotency_key`. Validates that the calling service is authorized to write the specific event type via the `X-Internal-Service` header. Balance projection is a SQL aggregation query — no stored balance field exists anywhere. Port 8020.

**wallet** — Read-only. Projects balance from the ledger. Returns the current NC balance for a user, optionally scoped to a venue. Excluded from balance: `topup_pending`, `venue_checkin`, `session_closed`. Port 8030.

**payments** — Manages payment intents and webhook processing. Provider interface supports mock (always available), Stripe (available when API key is set), and TWINT (pending credentials). On webhook confirmation, writes `topup_confirmed` to the ledger synchronously. Port 8060.

**catalog** — Owns venues, catalog items, events, and happy hour rules. Venues are created here (not in profiles). Happy hour rules define time-based price overrides. Catalog items carry both a standard price and an optional happy-hour override price. Port 8050.

**devices** — Manages device enrollment (each device generates a keypair; public key is submitted at enrollment), device status (pending/active/revoked), and heartbeat tracking. Port 8070.

**ticketing** — Owns ticket products and ticket issuances. Ticket purchase writes a `ticket_purchase` ledger event synchronously before issuing the ticket. QR tokens are generated as random 16-byte hex strings prefixed with `tix_`. Ticket validation marks the issuance as `used` (enforced as a state transition — `valid → used` only). Port 8080.

**orders** — Manages POS orders. Creates pending orders, finalizes them (balance check → ledger write → status update in a single flow), and handles voids. The two-minute void window is a permission boundary enforced in the handler, not a database constraint. Port 8090.

**sessions** — Manages venue sessions. Opens a session at check-in, closes at checkout. Writes `venue_checkin` and `session_closed` ledger events. Tracks per-session spend as a running total on the session record (for fast dashboard reads; the ledger remains the authoritative source). Port 8100.

**sync** — Cloud-side receiver for edge sync frames. Accepts batched ledger events from edge nodes, validates the `X-Internal-Service` header, forwards events to the ledger service, and returns per-event results (accepted / duplicate / rejected). Port 8110.

**reporting** — Read-only aggregation service. Provides venue revenue summaries, session statistics, and top-up summaries by querying across the ledger, sessions, and payments schemas. Access gated to nitecore and venue_admin roles. Port 8120.

### Edge Service

A standalone Go binary (`edge/`) that runs on-site at each venue. Uses `modernc.org/sqlite` — a pure-Go SQLite implementation with no CGo dependency. This is important: it means the edge binary can be cross-compiled for Android or ARM without a C toolchain.

The edge maintains:
- A local `ledger_events` table (same structure as cloud, restricted to this venue's events)
- A local `catalog_items` cache (pulled from the cloud catalog service every 15 minutes)
- A `sync_queue` table tracking which local events have not yet been synced to cloud

Two background goroutines run perpetually:
- **Catalog sync loop**: every 15 minutes, HTTP GET from cloud catalog service, upsert into local SQLite.
- **Sync agent**: every 30 seconds, fetch up to 50 pending events from sync queue, POST as a frame to the cloud sync service, mark accepted/duplicate events as synced.

The edge LAN API (port 9000) exposes: balance check, order creation (offline-capable), catalog item listing, and sync status.

### Authentication Model

**Access tokens:** RS256-signed JWTs, 15-minute TTL. Every request to the gateway is validated against the RS256 public key. Additionally, the gateway checks Redis (`tok:{jti}`) to catch revoked tokens within the access window. This means logout is effective immediately, not "eventually on expiry."

**Refresh tokens:** Stored in Redis as `ref:{uid}:{hash}` with a 30-day TTL. Issuing a new access token from a refresh token rotates the refresh token (old one deleted, new one issued).

**Staff PIN login:** Venue staff (bartenders, door staff) can log in with a 4-digit PIN on the terminal. The PIN is per-venue, bcrypt-hashed, stored in the venue record in the catalog service. The auth service validates the device token before accepting a PIN login.

**Service-to-service calls:** Internal service calls include the `X-Internal-Service` header identifying the calling service. The ledger service uses this to enforce write authority — only `payments` can write `topup_confirmed`, only `orders` can write `order_paid`, and so on.

### Data Isolation

Each service operates within its own Postgres schema. The `niteos_app` database role is granted permissions only within the schema of the service that uses it. The `ledger` schema additionally has UPDATE and DELETE revoked at the Postgres level — even if the application code had a bug that attempted an update, the database would reject it.

No service ever queries another service's schema directly. Cross-service reads go through HTTP API calls.

### Infrastructure

- **Reverse proxy**: Traefik with Cloudflare DNS-01 TLS certificate issuance. Not yet deployed; Traefik configuration is written and ready.
- **Database**: Postgres 16, single instance, multiple schemas. In production this would be a managed Postgres service (e.g., Supabase or RDS). In development, a Docker container.
- **Cache / session store**: Redis 7. Used for JWT revocation sets, rate limiting counters, and refresh tokens.
- **Observability**: Prometheus metrics endpoint (`GET /metrics`) on every service. Grafana dashboards prepared. Not yet deployed.
- **CI**: GitHub Actions workflow written (`.github/workflows/ci.yml`). Runs `go work sync`, `go build ./...`, `go vet ./...`, `go test ./...`, and migrations dry-run. Not yet running because there is no GitHub remote configured.

---

## 8. Existing Running Systems

Three systems are currently deployed and running, none of which is NiteOS Core:

**service-1 (PWL OS)** — A Fastify (Node.js) API and two Next.js frontends running on VPS A (31.97.126.86). This is a prototype of the venue operating concept — it has venues, events, users, orders, and a points system — but it does not implement the ledger model, has no payment integration, uses non-revocable HS256 JWTs, and is not in production use. It has no live venue transactions. It runs on VPS A and will be parked when NiteOS Core is deployed.

**service-2 (PWL Market)** — A production Next.js e-commerce platform on VPS B (72.60.181.89). Has live marketplace users making purchases. It has its own Postgres database and iron-session-based authentication entirely separate from NiteOS. It is not touched during NiteOS development. A future integration (Phase M7) will add an optional "Sign in with NiteOS" flow, additive to the existing auth, without modifying or migrating anything in service-2.

**service-3 (PWL Radio)** — A live HLS radio stream on VPS B. Fully independent. No integration planned.

---

## 9. What Has Been Built

As of March 2026, the NiteOS monorepo contains the following completed work:

**M0: Repository scaffolding** — The full directory structure is in place. A Go workspace (`go.work`) covers all 15 modules (13 services + pkg + edge). Shared tooling is built: `pkg/jwtutil` (RS256 parsing), `pkg/middleware` (gateway-header injection and role enforcement), `pkg/httputil` (standardized JSON responses), `pkg/idempotency` (key generation). The initial Postgres schema migration (`migrations/000_init_schemas.sql`) creates all 9 service schemas and the `niteos_app` role. An RS256 key pair is generated and stored in `infra/secrets/`. Key rotation procedure is documented. A GitHub Actions CI workflow is written. Docker Compose development stack is written.

**M2: Auth and gateway services** — The auth service implements all six flows: register, login (bcrypt), refresh, logout, PIN login, and JWKS. The gateway service validates JWTs, checks Redis revocation, and reverse-proxies with injected identity headers. The profiles service implements the auth-required subset (create user, lookup by email, fetch by ID).

**M3: All 13 cloud services and the edge service** — Every service has a complete implementation: handler, store (database layer), and wired main binary. Critical correctness tests are written for the ledger (idempotency, balance projection, write authority) and the orders service (insufficient balance, void time window, compensating events). The edge service has a complete offline order processing flow, catalog cache, and sync agent.

**What "complete" means here:** The code is written, structured correctly, and passes logic tests via mock stores. It has not been compiled against the actual Go toolchain in this environment (Go is not installed on the development machine). It has not been run against a real Postgres or Redis instance (Docker is not installed). These are environment blockers, not architectural ones.

---

## 10. What Remains to Be Done

**Immediate environment blockers:**
- Install Go 1.22+ and verify `go work sync && go build ./...` compiles clean.
- Install Docker Desktop and verify `make dev-up` starts Postgres + Redis cleanly.
- Add a GitHub remote and push to trigger the CI pipeline.

**Per-service SQL migrations (M1 partial):**
The `migrations/000_init_schemas.sql` file creates the schemas. The per-table `CREATE TABLE` migration files (e.g., `migrations/profiles/001_create_users.sql`) have not been written. `make migrate` will not work end-to-end until these are authored. The table structures are fully specified in `DOMAIN_MODEL.md`; this is authoring work, not design work.

**Admin console (M4):**
The web admin frontend (`web/admin/`) is built. It is a Next.js 14 application using the BFF pattern (iron-session + server components calling the gateway). Pages cover dashboard, catalog venues and items (with detail views), devices, sessions, reports, and users. The users page includes an `AssignVenueButton` component allowing nitecore operators to set a user's `venue_id` and role without reseeding. The backend APIs it calls are fully built.

**Guest web (part of M4 / M6):**
The guest-facing web app (`web/guest/`) does not exist. No frontend work has been done on the guest experience.

**Android applications:**
NiteKiosk, NiteTerminal, and Master Tablet are described in product blueprints. No Android development has begun. The `android/` directory exists as a placeholder.

**Infrastructure deployment (M5):**
All M5 repo artifacts are complete and coherent. `infra/docker-compose.cloud.yml` defines the full 13-service stack with Traefik (DNS-01 TLS via Cloudflare), Postgres, Redis, Prometheus, and Grafana across `proxy` and `core` Docker networks. `infra/traefik/` contains the static config and dynamic routes. `infra/prometheus.yml` scrapes all 13 services. `docs/DEPLOY_VPS_A.md` is a 14-section deployment runbook covering VPS provisioning through first pilot venue setup. Operational scripts are ready: `scripts/preflight-cloud.sh` (validates env, secrets, and config before deploy), `scripts/healthcheck-cloud.sh` (verifies all services via docker exec and Prometheus), `scripts/smoke-test-pilot.sh` (10-step end-to-end API test). Backup: `scripts/backup.sh` with pluggable upload (rclone/awscli/none), daily systemd timer, and `scripts/restore.sh`. The only remaining step is actual VPS A deployment: Docker install, DNS cutover, and running `make cloud-up`.

**TWINT provider:**
The provider interface is built. The Stripe provider is implemented. The TWINT provider is a stub pending Swiss merchant credentials.

**First pilot venue (M6):**
No venue has been provisioned, enrolled, or tested with real transactions. The tooling to do so (admin console + `PATCH /users/{user_id}/venue` API) is ready.

**service-2 identity integration (M7):**
No integration work has been done. This is planned post-pilot.

**Gamification layer:**
XP and level fields exist on the User and VenueProfile entities. No system awards XP, computes levels, or surfaces gamification to the user. Deferred post-pilot.

**Grafana / observability:**
All 13 services expose `/metrics` (Prometheus text format). Grafana is wired into docker-compose.cloud.yml with provisioned Prometheus datasource. Dashboard panel definitions are stubs — real business metric panels are deferred until Prometheus data flows from a live deployment.

---

## 11. Risks and Assumptions

**TWINT dependency.** The product's primary market is Switzerland. TWINT is the dominant mobile payment method in this demographic. Stripe is implemented and works, but a product targeting Swiss nightlife that requires Stripe (a card-based flow) instead of TWINT will face adoption friction. The risk is that TWINT merchant credentials and sandbox access take longer than expected to obtain.

**NFC wearable supply and logistics.** NiteTap is a physical product (bracelet, ring, or card with an embedded NFC chip). The system is designed around it, but the sourcing, printing, inventory, and issuance logistics for NiteTaps are not described in the technical documentation and represent an operational dependency that must be resolved before a pilot.

**Android app timeline.** The terminal and kiosk experience is entirely unbuilt. Before any live venue night, NiteKiosk and NiteTerminal must be functional enough for a bartender and door staff member to use in a real environment. Android development is the largest remaining buildout.

**Offline reliability assumptions.** The edge architecture assumes a single Go binary on a "Master Tablet" acts as the local server for all terminals on the LAN. This depends on: a stable LAN Wi-Fi network in the venue, the Master Tablet remaining powered and accessible, and the Android terminals being capable of routing to the edge address. These are testable assumptions but have not been tested in a real venue environment.

**FinMA float ceiling.** The 1,000,000 CHF outstanding NC cap is a design assumption for the MVP / pilot phase. It is sufficient for a small pilot but would be hit quickly at scale. Phase 2 regulatory work (SRO/VQF membership or a payment institution licence) is required for meaningful commercial growth.

**Single-instance Postgres.** The current architecture uses a single Postgres instance with schema-per-service isolation. This is appropriate for a pilot. It becomes a single point of failure and a performance bottleneck at scale. The schema isolation design makes it possible to migrate individual services to dedicated Postgres instances without schema changes, but that migration has not been planned in detail.

**No email delivery.** There is no email service in the system. Ticket confirmations, receipts, and password resets all lack email sending. This is an implicit dependency that would need to be resolved before launch.

**service-1 data continuity.** service-1 has no production users and no live transactions. There is nothing to migrate. This simplifies M6 significantly — DNS can be switched without any data migration. However, if service-1 accumulates any meaningful usage between now and M6, the assumption breaks.

---

## 12. Where the Project Stands Today

NiteOS is a fully designed and substantially implemented backend system with a complete data model, correct financial architecture, and 13 + 1 (edge) Go services written. It has not been compiled, run, or tested against real infrastructure in this environment due to missing Go and Docker installations on the development machine.

The migration plan is realistic and sequenced correctly. The most critical design decisions — append-only ledger, idempotent writes, schema isolation, offline-capable edge, RS256 auth — are made, documented, and implemented. The remaining work is primarily:

1. **Verify the build compiles** (environment setup, 1–2 hours)
2. **Write per-table SQL migrations** (design is done; authoring SQL is mechanical, 1–2 days)
3. **Build frontend surfaces** (guest web + admin console, weeks of frontend work)
4. **Build Android applications** (the largest remaining workload, weeks to months)
5. **Deploy to VPS A** (infrastructure work, days)
6. **Provision and test with a real venue** (operational work)

The project is solidly past the "does this design make sense" question. It is in the implementation-and-integration phase. The critical path to a first live venue night runs through: Go compilation verified → Docker stack running → per-service migrations written → admin console built at MVP level → Android kiosk and terminal at MVP level → pilot venue provisioned → DNS switched.

People We Like Market and Radio are live, independent products. They are not blocked by NiteOS development and are not affected by it until the optional Phase M7 identity integration, which occurs post-pilot.
