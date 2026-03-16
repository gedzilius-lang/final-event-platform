# SERVICE_MAP.md

Maps every running service, its exposed interfaces, and the relationships (real and intended) between them.

---

## Current Live Topology

```
Internet
    |
    +--- Cloudflare (DNS proxy, partial caching)
    |
    +--- VPS A: 31.97.126.86  (NiteOS / PWL OS)
    |         |
    |         +-- nginx (port 80)
    |               |-- os.peoplewelike.club      --> Next.js OS frontend  (container)
    |               |-- admin.peoplewelike.club   --> Next.js Admin frontend (container)
    |               |-- api.peoplewelike.club     --> Fastify API (container :4000)
    |                                                   |
    |                                                   +-- PostgreSQL (container, internal)
    |
    +--- VPS B: 72.60.181.89  (Market + Radio)
              |
              +-- nginx (host, port 80/443)
                    |-- market.peoplewelike.club  --> Next.js Market app (container :3101)
                    |                                   |
                    |                                   +-- PostgreSQL (container, internal)
                    |
                    |-- radio.peoplewelike.club   --> nginx:alpine (container :8080)
                    |   stream.peoplewelike.club         |-- /hls/current/index.m3u8
                    |   ingest.peoplewelike.club         |-- /api/status
                    |                                    |-- /api/nowplaying
                    |                                    |-- / (Video.js player)
                    |
                    |   RTMP :1935 -----------> radio-rtmp container
                    |                               |-- application live (auth via radio-rtmp-auth :8088)
                    |                               |-- application autodj (localhost only)
                    |
                    +-- av.peoplewelike.club      --> Unrelated static site (do not touch)
```

---

## Service Interfaces

### service-1: PWL OS API (`api.peoplewelike.club`)

| Route Group | Endpoints | Auth Required | Notes |
|-------------|-----------|---------------|-------|
| Auth | `POST /auth/login`, `POST /auth/pin` | No | JWT issued on success |
| Identity | `GET /me`, `GET /me/profile`, `GET /me/history` | JWT | User + session state |
| Guest ops | `POST /guest/checkin`, `POST /guest/checkout` | No | uid_tag optional |
| Venues | `GET /venues`, `POST /venues`, `GET /venues/:id/*` | Mixed | Admin creates, staff reads |
| Inventory | `GET/POST/PATCH /venues/:id/inventory` | JWT (staff) | Stock levels |
| Menu | `GET /venues/:id/menu`, `POST/PATCH /menu-items` | JWT | POS catalog |
| Orders | `POST /orders` | JWT (BAR role) | Decrements inventory |
| Events | `GET /events`, `POST /events` | Mixed | Public read, admin write |
| Quests | `GET/POST /quests`, `POST /quests/:id/complete` | JWT | XP + NC rewards |
| Notifications | `GET /notifications` | JWT | Role-targeted messages |
| Automation | `GET/POST /automation-rules` | JWT (admin) | Trigger-condition-action rules |
| Logs | `GET /logs` | JWT (admin) | Operational event log |
| Analytics | `GET /analytics` | JWT (admin) | Named event stream |
| Vendors | `GET/POST /vendors`, `GET /vendors/:id/products` | Mixed | Marketplace stub |
| System | `GET /health`, `GET /status`, `GET /config` | Mixed | Ops monitoring |

### service-1: PWL OS Frontend (`os.peoplewelike.club`)

| Surface | Function |
|---------|----------|
| Event feed | Lists upcoming events from API |
| Event detail | Single event page |
| Radio embed | Iframe pointing to `RADIO_IFRAME_SRC` (radio.peoplewelike.club) |
| Auth flow | Sign in (email/password or demo) |

### service-1: PWL Admin Frontend (`admin.peoplewelike.club`)

| Surface | Function |
|---------|----------|
| Venue management | Create venues, set PIN |
| Event management | Create/edit events |
| Inventory | View and adjust stock |
| Logs | View operational log |
| Staff demo | "Sign In Demo" testing shortcut |

### service-2: PWL Market (`market.peoplewelike.club`)

| Route | Surface | Auth |
|-------|---------|------|
| `/` | Homepage: active drop hero + featured products | Public |
| `/market` | Browse products with tag filters + taste sliders | Public |
| `/p/[slug]` | Product detail | Public |
| `/v/[slug]` | Vendor atelier page | Public |
| `/drop/[slug]` | Editorial drop page | Public |
| `/login` | Email + password login | Public |
| `/apply` | Vendor application form | Public |
| `/invite/[token]` | Vendor password setup | Public (token gated) |
| `/vendor/*` | Vendor dashboard (product CRUD, images, orders) | Vendor session |
| `/admin/*` | Admin dashboard (vendors, products, drops, orders) | Admin session |

### service-3: Radio (`radio.peoplewelike.club`)

| Interface | Protocol | Notes |
|-----------|----------|-------|
| `/hls/current/index.m3u8` | HTTP GET | Stable public HLS stream |
| `/api/nowplaying` | HTTP GET | JSON: title, artist, mode |
| `/api/status` | HTTP GET | JSON: source (live/autodj), seq, updated |
| `/` | HTTP GET | Video.js web player |
| `rtmp://ingest.peoplewelike.club:1935/club/live` | RTMP | Live DJ ingest, stream key auth |
| RTMP stats (internal) | HTTP `:8089` | Used by switch daemon, not public |

---

## Inter-Service Relationships (Current State)

### What is connected today

| From | To | How | Notes |
|------|----|-----|-------|
| service-1 OS frontend | service-3 Radio | iframe embed | `RADIO_IFRAME_SRC` env var |
| service-1 OS frontend | service-1 API | REST over internal Docker network | Direct container link |
| service-1 Admin frontend | service-1 API | REST over internal Docker network | Direct container link |

### What is NOT connected but should be

| Missing Link | Why It Matters |
|--------------|----------------|
| service-1 API → any payment provider | Wallet top-up has no real money path |
| service-1 API → service-2 Market | Vendor products in OS DB are a duplicate of service-2 |
| service-3 Radio → service-1 OS | Now playing data not surfaced in OS beyond iframe |
| Any service → email delivery | Vendor invites, ticket confirmations, wallet receipts all lack email sending |
| Any service → mobile/Android terminal | No edge/kiosk layer exists yet |
| Any service → SQLite edge node | No offline-capable edge service exists |

---

## Intended Future Topology (from architecture-notes.md)

```
Internet
    |
    +-- Cloudflare
         |
         +-- api.peoplewelike.club  (Go API gateway)
         |       |
         |       +-- auth service
         |       +-- profiles service
         |       +-- ledger service (append-only, Postgres)
         |       +-- wallet service (projections from ledger)
         |       +-- ticketing service
         |       +-- orders service
         |       +-- catalog service
         |       +-- devices service
         |       +-- reporting service
         |       +-- payments service (TWINT primary, Stripe optional)
         |       |
         |       +-- Postgres (cloud)
         |       +-- Redis (sessions, cache, rate limits)
         |       +-- NATS (optional async)
         |       +-- Traefik (reverse proxy)
         |       +-- Grafana (observability)
         |
         +-- os.peoplewelike.club   (Next.js guest web)
         +-- admin.peoplewelike.club (Next.js admin console)
         |
         +-- radio.peoplewelike.club (keep separate)

    Venue LAN
         |
         +-- Edge Node (Master Tablet or NiteBox)
               |-- SQLite hot ledger
               |-- local order capture
               |-- local sync agent
               |-- device coordination
               |
               +-- Android terminals (via LAN Wi-Fi)
                     |-- NiteKiosk (bartender, kiosk mode)
                     |-- NiteTerminal (door/security)
                     |-- Master Tablet (admin)
```

---

## Key Observations

1. **Two separate VPS** are currently in use. The architecture notes require consolidation onto one primary VPS (`31.97.126.86`) for NiteOS core, with radio remaining on `72.60.181.89`.

2. **service-2 (Market) and service-3 (Radio) share a VPS** (`72.60.181.89`) but have no integration with each other or with service-1.

3. **No shared auth** exists between any of the three services. A user logged into Market cannot access the OS or vice versa.

4. **No edge layer exists** anywhere. The entire offline-capable venue operating system described in architecture-notes.md is not yet built.

5. **service-1 is the closest thing to NiteOS core** but lacks: event-sourced ledger, payment integration, device enrollment, proper sync, and Go rewrite.

6. **service-2 (Market) is architecturally separate** from the venue ops product. It is a public e-commerce tool, not a venue operating system.

7. **service-3 (Radio) is fully independent** and should remain so per the architecture direction.
