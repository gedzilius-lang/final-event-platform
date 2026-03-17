# SERVICE_MAP.md

Maps every running service, its exposed interfaces, and the relationships (real and intended) between them.

---

## Current Live Topology

Last verified: 2026-03-16

```
Internet
    |
    +--- Cloudflare (DNS proxy)
    |
    +--- NiteOS VPS: 31.97.126.86  (NiteOS Cloud Core — TARGET, not yet deployed)
    |         |
    |         +-- nginx (host, port 80/443) — ingress configured, routes to Traefik:
    |               |-- api.peoplewelike.club     --> Traefik → gateway (not yet live)
    |               |-- admin.peoplewelike.club   --> Traefik → admin-web (not yet live)
    |               |-- grafana.peoplewelike.club --> Traefik → Grafana (not yet live)
    |               |-- traefik.peoplewelike.club --> Traefik dashboard (not yet live)
    |         |
    |         +-- /opt/niteos: ABSENT (not yet deployed)
    |         +-- Docker: installed (v29.2.1), no NiteOS containers running
    |         +-- SSH: hardened (key-only, fail2ban, UFW)
    |         +-- Swap: 4 GB
    |
    +--- Radio VPS: 72.60.181.89  (Radio + Market — stable, do not touch)
              |
              +-- nginx (host, port 80/443)
                    |-- market.peoplewelike.club  --> pwl-market-app (container :3101)
                    |                                   +-- PostgreSQL (container, internal)
                    |-- more.peoplewelike.club    --> pwl-more-app (container :3100)
                    |                                   +-- PostgreSQL (container, internal)
                    |-- radio.peoplewelike.club   --> radio-web nginx:alpine (container :8080)
                    |   stream.peoplewelike.club         |-- /hls/current/index.m3u8
                    |   ingest.peoplewelike.club         |-- /api/status, /api/nowplaying
                    |
                    |   RTMP :1935 -----------> radio-rtmp container
                    |                               |-- auth via radio-rtmp-auth (:8088)
                    |                               |-- autodj (localhost)
                    |
                    +-- av.peoplewelike.club      --> Unrelated static site (do not touch)
```

**NiteOS VPS state summary (2026-03-16):**
- Docker 29.2.1 installed and running
- nginx routes :443 → Traefik for NiteOS subdomains (ingress only; no backend services yet)
- /opt/niteos does not exist — repo not yet cloned, stack not deployed
- No Postgres, no Redis, no JWT keys, no cloud.env on this machine
- 35 GB disk free, 4 GB swap, SSH hardened

**Radio VPS state summary (2026-03-16):**
- 8 Docker containers running, all healthy
- pwl-market-backup.timer active (daily backup)
- service-1 (the old Fastify prototype) is NOT running — it was never deployed here

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
| service-2 Market | service-3 Radio | none (separate stacks) | No integration |
| guest-web | service-3 Radio | iframe embed | guest-web built (not yet deployed) |

service-1 is not deployed anywhere and has no live inter-service connections.

### What is NOT yet connected but will be

| Missing Link | Why It Matters |
|--------------|----------------|
| NiteOS gateway → any service | NiteOS stack not yet deployed on NiteOS VPS |
| guest-web → NiteOS gateway | guest-web built; blocked on NiteOS VPS deployment |
| Android terminals → edge node | Android apps not yet built |
| Edge node → cloud sync service | Depends on NiteOS VPS deployment |
| NiteOS payments → TWINT | Waiting for Swiss business account credentials |
| service-2 Market → NiteOS OAuth | M7 — shared identity, deferred post-pilot |

---

## Target Topology (after M5+M6 deployment)

```
Internet
    |
    +-- Cloudflare
         |
         +-- NiteOS VPS: 31.97.126.86
         |    |
         |    +-- nginx (host :80/:443)
         |         |-- api.peoplewelike.club   → Traefik → gateway :8000
         |         |-- admin.peoplewelike.club → Traefik → admin-web :3001
         |         |-- os.peoplewelike.club    → Traefik → guest-web :3000
         |         |-- grafana.peoplewelike.club → Traefik → Grafana :3100
         |
         |    +-- gateway :8000
         |         +-- auth :8010
         |         +-- profiles :8020
         |         +-- ledger :8030       (append-only Postgres, no UPDATE/DELETE)
         |         +-- wallet :8040       (balance projections from ledger)
         |         +-- payments :8050     (Stripe + TWINT, webhook receiver)
         |         +-- ticketing :8060
         |         +-- orders :8070
         |         +-- catalog :8080
         |         +-- devices :8090
         |         +-- sessions :8100
         |         +-- reporting :8110
         |         +-- sync :8120         (receives edge sync frames)
         |
         |    +-- Postgres :5432          (single instance, schema-per-service)
         |    +-- Redis :6379             (sessions, rate limits, wallet cache)
         |    +-- Traefik                 (DNS-01 TLS via Cloudflare)
         |    +-- Prometheus + Grafana :3100
         |
         +-- Radio VPS: 72.60.181.89     (stable — do not modify)
              +-- radio.peoplewelike.club  (keep separate, embedded in guest-web via iframe)
              +-- market.peoplewelike.club (M7: additive shared identity only)

    Venue LAN (per venue)
         |
         +-- Edge Node (Master Tablet, Go binary + SQLite)
               |-- LAN API :9000
               |-- local ledger, catalog cache, sync queue
               |
               +-- Android terminals (via isolated Ubiquiti VLAN)
                     |-- NiteKiosk (bartender, kiosk mode)
                     |-- NiteTerminal (door/security)
                     |-- Master Tablet (admin + edge host)
```

---

## Key Observations

1. **Two VPS machines** are in use. Role assignment is fixed:
   - NiteOS VPS (31.97.126.86): clean dedicated NiteOS runtime. Not yet deployed.
   - Radio VPS (72.60.181.89): stable runtime for radio, market, more. Do not touch.

2. **service-2 (Market) and service-3 (Radio)** are live on Radio VPS and have no integration with NiteOS. M7 adds additive-only shared identity to Market. Radio integration is limited to guest-web iframe embed.

3. **No shared auth** exists yet. Planned: NiteOS auth becomes the OAuth2 provider for Market (M7, post-pilot).

4. **NiteOS Cloud Core is built but not deployed.** All 13 Go services + edge + admin-web + guest-web exist in this repo and build clean. They are not yet running on any VPS.

5. **service-1 was a prototype** (Fastify API + Next.js). It was never deployed to production. It is reference material only and will not be migrated.

6. **The edge layer is implemented** in `edge/` (Go binary + SQLite). Not yet deployed to any venue hardware.

7. **service-3 (Radio) is fully independent** and must remain so. It runs on the Radio VPS and will not be touched by any NiteOS deployment activity.
