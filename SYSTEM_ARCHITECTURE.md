# SYSTEM_ARCHITECTURE.md

The complete system architecture for NiteOS. Every service, every communication path, every deployment boundary.

---

## Architectural Overview

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              INTERNET / CLOUDFLARE                               │
└──────────────────────────────┬──────────────────────────────────────────────────┘
                               │  HTTPS / DNS-01 TLS
                               ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         VPS A — 31.97.126.86 (NiteOS Core)                      │
│                                                                                  │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                          Traefik (Reverse Proxy)                        │   │
│   │          api. / os. / admin.  .peoplewelike.club                        │   │
│   └──────────────────────┬──────────────────────────────────────────────────┘   │
│                          │ internal routing                                      │
│   ┌──────────────────────▼──────────────────────────────────────────────────┐   │
│   │                        gateway (Go)   :8000                             │   │
│   │       auth check · rate limit · route · API composition                 │   │
│   └──┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┘   │
│      │      │      │      │      │      │      │      │      │      │           │
│   ┌──▼─┐ ┌──▼─┐ ┌──▼──┐ ┌─▼──┐ ┌─▼──┐ ┌─▼──┐ ┌─▼──┐ ┌─▼──┐ ┌─▼──┐ ┌─▼──┐   │
│   │auth│ │prof│ │ledgr│ │wllt│ │pay │ │tick│ │ordr│ │ctlg│ │dev │ │sess│   │
│   │:81 │ │:82 │ │:83  │ │:84 │ │:85 │ │:86 │ │:87 │ │:88 │ │:89 │ │:90 │   │
│   └──┬─┘ └──┬─┘ └──┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘ └─┬──┘   │
│      │      │      │      │      │      │      │      │      │      │           │
│   ┌──▼──────▼──────▼──────▼──────▼──────▼──────▼──────▼──────▼──────▼──────┐   │
│   │                      PostgreSQL :5432  +  Redis :6379                   │   │
│   └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │  reporting :91  ·  sync :92  ·  guest-web :3000  ·  admin-web :3001    │   │
│   └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │                     Grafana :3100  ·  (Loki · Prometheus later)         │   │
│   └─────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│                     VPS B — 72.60.181.89 (Radio + Market)                       │
│                                                                                  │
│   radio.peoplewelike.club  ──►  radio stack (nginx-rtmp · switch · autodj)     │
│   market.peoplewelike.club ──►  Next.js market app + Postgres                  │
└─────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────┐
│                          VENUE (per-venue, on-premises)                          │
│                                                                                  │
│   Master Tablet (Android)                                                        │
│   ├── edge service (Go binary, SQLite)  ← LAN API :9000                         │
│   └── admin-web (embedded webview) OR android-master app                        │
│                                                                                  │
│   ┌────────── Ubiquiti UniFi VLAN (NiteOS isolated) ──────────┐                 │
│   │                                                            │                 │
│   │  NiteKiosk ×1-3 (Android)    NiteTerminal ×1 (Android)   │                 │
│   │  → edge :9000 (LAN)          → edge :9000 (LAN)          │                 │
│   │  → gateway :8000 (fallback)  → gateway :8000 (fallback)  │                 │
│   └────────────────────────────────────────────────────────────┘                │
│                                                                                  │
│   Edge ──sync──► cloud sync service :92 (when internet available)               │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## Cloud Services (Go Microservices)

Each service is a Go 1.22+ binary. Each owns its own Postgres schema. Each communicates with other services via internal HTTP or gRPC (HTTP/JSON in Phase 1; gRPC optional in Phase 2). No service shares a database table with another service.

### 1. gateway (`:8000`)
Public entry point for all external traffic.

Responsibilities:
- Validates auth tokens on every request (calls auth service via Redis token cache)
- Routes requests to upstream services
- Rate limiting per IP and per user
- Request/response logging
- API composition for endpoints that aggregate multiple services
- Returns 503 with cached last-known data when upstream is degraded

Does NOT: hold business logic, own any data, trust request headers from external clients.

### 2. auth (`:8010`)
All identity operations.

Responsibilities:
- Email + password login → issues access token (JWT, 15 min) + refresh token (opaque, stored in Redis, 30 days)
- Venue PIN login → issues device-scoped JWT for staff terminals
- Device enrollment auth → validates device credentials, issues device JWT (90 days)
- Access token validation (used by gateway, cached in Redis for performance)
- Refresh token exchange
- Session revocation (delete Redis entry)
- Rate limiting on login endpoints (5 req/min per IP)
- Password hash with bcrypt (cost 12)

Token payload (access JWT):
```json
{
  "uid": "usr_abc123",
  "role": "guest|venue_admin|bartender|door_staff|nitecore",
  "venue_id": "ven_xyz789",
  "device_id": "dev_def456",
  "session_id": "ses_ghi012",
  "iat": 1700000000,
  "exp": 1700000900
}
```

Redis keys:
- `tok:{jti}` → `{uid}:{session_id}` — active token record (TTL = token expiry)
- `rev:{uid}:{session_id}` → `1` — revocation flag
- `dev:{device_id}` → `{venue_id}:{role}:{enrolled_at}` — device session

### 3. profiles (`:8020`)
User identity and venue membership.

Responsibilities:
- User CRUD: create, read, update (email, display name, preferences)
- Global XP and level tracking
- Venue profile: per-user per-venue local XP, local level
- NiteTap registration: associate NFC UID to user account
- NiteTap lookup: resolve NFC UID → user_id (used by terminals)
- Anonymous session management (NiteTap UIDs not linked to a user)

### 4. ledger (`:8030`)
The most critical service. Append-only financial event store.

Responsibilities:
- Append ledger events (INSERT only, no UPDATE, no DELETE)
- Idempotency enforcement via `idempotency_key` unique constraint
- Balance projection: compute current NiteCoin balance from event history for a user/venue
- Event validation: reject malformed or out-of-sequence events
- Reconciliation: compare ledger totals vs payment provider totals
- Audit queries: event history for a user, venue, or device

Hard rules enforced in code:
- No UPDATE or DELETE statements in any query in this service
- `topup_confirmed` events can only be written by the payments service (validated by calling service identity header)
- Every event must have a non-null `idempotency_key`
- Balance projections are computed at query time or from a Redis-cached projection (never stored in Postgres)

Event schema:
```
ledger_events (
  event_id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  event_type      text NOT NULL,        -- topup_pending | topup_confirmed | order_paid |
                                        --   refund_created | bonus_credit | ticket_purchase |
                                        --   venue_checkin | session_closed
  user_id         uuid NOT NULL,
  venue_id        uuid,
  device_id       uuid,
  amount_nc       integer NOT NULL,     -- positive = credit, negative = debit
  amount_chf      numeric(10,2),        -- original fiat if applicable
  reference_id    uuid,                 -- order_id, payment_intent_id, ticket_id etc.
  idempotency_key text UNIQUE NOT NULL,
  occurred_at     timestamptz NOT NULL DEFAULT now(),
  synced_from     text,                 -- 'cloud' | 'edge:{venue_id}'
  metadata        jsonb
)
```

### 5. wallet (`:8040`)
Balance views and voucher semantics.

Responsibilities:
- Serve wallet balance (calls ledger projection, caches in Redis with 5s TTL)
- Pending vs confirmed balance distinction
- Bonus credit application logic (venue-configured bonus rules)
- Top-up state: pending → confirmed lifecycle
- Wallet history (paginated ledger event list for a user)

Does NOT own any Postgres tables. All state derived from ledger or cached in Redis.

### 6. payments (`:8050`)
Payment provider abstraction.

Responsibilities:
- CreateTopupIntent(amount_chf, user_id, venue_id) → intent_id + provider redirect URL
- Confirm/Authorize(intent_id) → validates provider response
- Capture(intent_id) → triggers `topup_confirmed` ledger event via ledger service
- Refund(ledger_event_id, amount_nc) → creates `refund_created` ledger event, triggers provider refund
- VerifyWebhook(provider, signature, payload) → validates incoming callback
- Reconcile(time_window) → diff ledger totals vs provider settlement report

Provider implementations (behind internal interface):
- TWINT: via PSP/aggregator (SIX Payment Services or equivalent)
- Stripe: direct integration

Postgres owns: `payment_intents` table only.

### 7. ticketing (`:8060`)
Ticket lifecycle.

Responsibilities:
- Ticket product definitions (title, price_chf, nc_included, capacity, event linkage)
- Ticket purchase: creates ledger event via ledger service, issues ticket
- QR token generation (HMAC-signed, single-use)
- Ticket validation: verify QR token, mark redeemed
- Inventory tracking: available vs issued
- Ticket lookup by QR token (called by sessions service at check-in)

Postgres owns: `ticket_products`, `ticket_issuances`.

### 8. orders (`:8070`)
POS order lifecycle.

Responsibilities:
- Create order (venue_id, device_id, guest_session_id, items[], total_nc)
- Validate: check wallet has sufficient balance via wallet service
- Finalize: call ledger service to write `order_paid` event → receive ledger_event_id
- Store order with ledger_event_id reference
- Void order: call ledger service to write `refund_created` compensating event
- Order history per venue and per user

Hard rule: an order is only persisted after the ledger event is written. If ledger write fails, the order is not persisted. This is enforced transactionally.

Postgres owns: `orders`, `order_items`.

### 9. catalog (`:8080`)
Venue menu and pricing.

Responsibilities:
- Catalog item CRUD (Venue Admin only)
- Price updates (effective immediately on next sync to edge)
- Availability flags (active/inactive, sold-out)
- Inventory counts: decrement on order (via order service event), increment on manual restock
- Happy Hour rules: time-based price overrides
- Catalog snapshot: serialized catalog pushed to edge on change

Postgres owns: `catalog_items`, `venue_catalogs`, `happy_hour_rules`.

### 10. devices (`:8090`)
Hardware enrollment and management.

Responsibilities:
- Device enrollment workflow: device generates keypair, submits public key + venue_id + role
- Venue Admin approves enrollment (or auto-approved if within enrollment window)
- Per-device credentials: issue device JWT on enrollment
- Device heartbeat: receive and store last-seen timestamp
- Device status: online/offline/pending/revoked
- Device revocation: invalidate device JWT in Redis
- Kiosk trust state: enrolled → active → revoked lifecycle

Postgres owns: `devices`, `device_heartbeats`.

### 11. sessions (`:8100`)
Venue check-in and session tracking.

Responsibilities:
- Open venue session (triggered by check-in: NiteTap tap or QR ticket scan)
- Write `venue_checkin` ledger event via ledger service
- Live occupancy counter (Redis: `venue:{venue_id}:occupancy`)
- Close session (triggered by exit scan, end-of-night close, or Venue Admin action)
- Write `session_closed` ledger event
- Session history per user and per venue
- Capacity enforcement: reject check-in when venue is at capacity

Postgres owns: `venue_sessions`.

### 12. reporting (`:8110`)
Operational visibility.

Responsibilities:
- Venue dashboard: tonight's spend, top items, occupancy, device status
- Sync health view: unsynced event count and age per venue
- Device heartbeat dashboard
- End-of-night summary: close report, PDF export
- Post-event analytics: per-item velocity, staff performance, reconciliation status
- Nitecore HQ view: network-wide NC circulation, revenue accumulation, float totals

Does NOT own primary data. Reads from ledger, orders, sessions, devices, sync via internal APIs or materialized views. Postgres schema: reporting-specific materialized snapshots only.

### 13. sync (`:8120`)
Edge-to-cloud event ingestion.

Responsibilities:
- Receive sync frames from edge nodes (authenticated by device JWT)
- Validate frame checksum and sequence
- Decompose frame into ledger events
- Submit events to ledger service (idempotent — duplicates are safe)
- Update sync state: mark frame as received, processed, or failed
- Report sync lag to reporting service

Postgres owns: `sync_frames`, `sync_state`.

---

## Edge Service (Go Binary)

Deployed per venue, runs on Master Tablet (or NiteBox in future).

```
edge (Go binary, SQLite)  ← /opt/niteos-edge/
├── LAN API  :9000        ← NiteKiosk + NiteTerminal connect here
├── SQLite   /data/edge.db
├── sync agent            ← flushes queue to cloud sync service
└── config   /etc/edge/config.toml
```

LAN API endpoints (internal, device-JWT authenticated):
- `POST /order` — create and finalize order (writes to SQLite ledger)
- `POST /checkin` — open venue session (writes to SQLite)
- `POST /checkout` — close venue session
- `GET  /catalog` — return current catalog snapshot
- `GET  /wallet/{niteTap_uid}` — return current balance projection from SQLite ledger
- `POST /device/heartbeat` — device reports health
- `GET  /health` — edge health check

SQLite schema mirrors cloud ledger schema exactly. Balance projection runs on SQLite.

Sync agent behaviour:
- Runs every 30 seconds when internet is available
- Bundles unsent ledger events into sync frames
- POSTs frames to cloud `sync` service
- On 200 OK: marks events as synced in SQLite
- On failure: retries with exponential backoff (cap 5 min)
- Never deletes unsynced events

---

## Frontend Services

### guest-web (Next.js 14, `:3000`)
Domain: `os.peoplewelike.club`

- App Router, TypeScript
- BFF pattern: Next.js API routes act as auth proxy (exchange tokens for session cookies)
- Public pages (event feed, ticket purchase): SSR + ISR
- Authenticated pages (wallet, profile, Venue Mode): server-rendered with session
- Radio embed: `<iframe>` to `radio.peoplewelike.club`
- Calls cloud services via gateway only (never directly to microservices)

### admin-web (Next.js 14, `:3001`)
Domain: `admin.peoplewelike.club`

- App Router, TypeScript
- Server-side rendered, session-gated throughout
- Venue Admin and Nitecore HQ views (role-determined at render)
- Calls cloud services via gateway only

---

## Android Applications (Kotlin)

All three apps share:
- Device Owner / kiosk mode enforcement at OS level
- Per-device credentials stored in Android KeyStore (not filesystem)
- LAN-first: connect to edge service at pre-configured local IP/hostname
- Cloud fallback: switch to gateway if edge unreachable for > 3 seconds
- Connectivity monitor runs in background, switches routes silently

### android-kiosk (NiteKiosk)
Role: `bartender`
Primary API: edge `:9000/order`, `:9000/catalog`, `:9000/wallet/{uid}`
NFC: reads NiteTap UID (HCE or passive NFC tag read)
UI: kiosk POS — full screen, no system UI visible

### android-terminal (NiteTerminal)
Role: `door_staff`
Primary API: edge `:9000/checkin`, `:9000/checkout`
Scan: QR camera scan + NFC NiteTap read
UI: check-in confirmation, capacity counter, walk-in onboarding

### android-master (Master Tablet)
Role: `venue_admin`
Primary API: edge `:9000/*` + gateway `:8000/*` (both)
Hosts: edge service binary as a background service (via Android foreground service + Work Manager)
UI: full venue admin — menu, staff, devices, dashboard, close night

---

## Service Communication

### External (client → cloud)
All external traffic enters through Traefik → gateway → upstream service.
No microservice is exposed directly to the internet.

### Internal (service → service)
HTTP/JSON over Docker internal network in Phase 1.
Service discovery: Docker DNS names (e.g., `http://ledger:8030`).
No external DNS or load balancer for inter-service calls.

### Inter-service call patterns

| Caller | Callee | When |
|--------|--------|------|
| gateway | auth | Every authenticated request (token validation, cached in Redis) |
| payments | ledger | On verified payment callback → write `topup_confirmed` |
| orders | ledger | On order finalization → write `order_paid` |
| orders | wallet | Pre-flight balance check before writing order |
| sessions | ledger | On check-in → write `venue_checkin`; on close → write `session_closed` |
| ticketing | ledger | On ticket purchase → write `ticket_purchase` |
| wallet | ledger | Balance projection (Redis cache hit first) |
| reporting | ledger, orders, sessions, devices, sync | Aggregation reads |
| sync | ledger | Submit edge events on frame receipt |
| catalog | (none) | Catalog is read by edge at sync; no real-time service calls |
| devices | auth | On enrollment completion → notify auth to register device credentials |

### Ledger write authority
Only these services may write to the ledger:
- `payments` → `topup_pending`, `topup_confirmed`, `refund_created` (payment-originated)
- `orders` → `order_paid`, `refund_created` (POS-originated)
- `ticketing` → `ticket_purchase`
- `sessions` → `venue_checkin`, `session_closed`
- `payments` → `bonus_credit` (promotional NC grants — financial operation)
- `sync` → any event type (proxied from edge, with source tag `synced_from: edge:{venue_id}`)

No other service may write ledger events. Gateway cannot write ledger events. Admin UI cannot write ledger events directly — it calls the appropriate service.

---

## Deployment Layout

### VPS A (NiteOS Core)
```
/opt/niteos/
├── docker-compose.yml          ← single source of truth
├── .env                        ← secrets (not in git)
├── traefik/
│   └── traefik.yml
├── postgres/
│   └── init/                   ← per-service schema init scripts
├── redis/
│   └── redis.conf
└── grafana/
    └── dashboards/
```

All services run as Docker containers. Traefik auto-discovers via container labels.
Postgres: single instance, per-service schemas (schema-per-service isolation).
Redis: single instance, key namespace per service (`auth:*`, `wallet:*`, etc.).

### VPS B (Radio + Market)
```
/opt/radijas-v2/    ← radio (existing, unchanged)
/opt/pwl-market/    ← market (existing, may relocate to VPS A later)
```

### Venue (per-venue)
```
/opt/niteos-edge/
├── edge              ← Go binary
├── edge.db           ← SQLite database
├── config.toml       ← venue_id, cloud endpoint, LAN config
└── logs/
```

---

## Observability

### Phase 1 (pilot-required)
- Grafana dashboard on VPS A
- Alerts for: service down, payment callback failure, sync lag > 5 min, device offline > 10 min, edge unreachable
- Structured JSON logs from all Go services (written to stdout, captured by Docker)
- `/health` endpoint on every service (returns 200 + version + uptime)
- `/metrics` endpoint on every service (Prometheus format)

### Phase 2 (add post-pilot)
- Loki log aggregation
- Prometheus scrape all `/metrics` endpoints
- Per-venue sync health dashboard
- Device heartbeat timeline view
- Payment reconciliation alert (ledger ≠ provider)

---

## Network Ports (Reference)

| Service | Internal Port | Notes |
|---------|--------------|-------|
| Traefik | 80, 443 | Public-facing |
| gateway | 8000 | All external API traffic |
| auth | 8010 | Internal only |
| profiles | 8020 | Internal only |
| ledger | 8030 | Internal only |
| wallet | 8040 | Internal only |
| payments | 8050 | Internal only + provider webhooks via gateway |
| ticketing | 8060 | Internal only |
| orders | 8070 | Internal only |
| catalog | 8080 | Internal only |
| devices | 8090 | Internal only |
| sessions | 8100 | Internal only |
| reporting | 8110 | Internal only |
| sync | 8120 | Internal only + edge connections via gateway |
| guest-web | 3000 | Traefik → os.peoplewelike.club |
| admin-web | 3001 | Traefik → admin.peoplewelike.club |
| Grafana | 3100 | Internal only (access via SSH tunnel) |
| Postgres | 5432 | Internal only |
| Redis | 6379 | Internal only |
| Edge LAN API | 9000 | Venue LAN only, not internet-facing |
