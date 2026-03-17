# Infrastructure State

Canonical record of what has been built, deployed, and established.
Updated as work progresses. Use this file to orient any new session or handover.

---

## Machine Inventory

| Role | IP | OS | Purpose |
|------|----|----|---------|
| **NiteOS VPS** | `31.97.126.86` | Ubuntu 22.04 LTS | NiteOS Cloud Core runtime — **M6 live** |
| **Radio VPS** | `72.60.181.89` | Ubuntu 22.04 LTS | Radio + Market + More runtime (stable, do not touch) |

**VPS naming rule:** Do not use "VPS A / VPS B" labels — they were ambiguous. Use role names or IPs directly.

---

## NiteOS VPS — 31.97.126.86

### Current State — M6 Live (2026-03-17)

| Item | Status | Notes |
|------|--------|-------|
| OS | Ubuntu 22.04.5 LTS | Kernel 5.15.0-170-generic |
| Docker | v29.2.1, Compose v5.0.2 | Running, enabled at boot |
| SSH hardening | Done | Key-only root login, fail2ban, UFW |
| UFW firewall | Active | Allow: 22, 80, 443 |
| Traefik | **v3.6 running** | Direct on :80/:443; Docker provider + file provider |
| /opt/niteos | Present — branch `main` @ M6 | 20 containers healthy |
| cloud.env | Present | Real secrets, not in git |
| JWT keys | Present | `infra/secrets/jwt_{private,public}_key.pem` |
| acme.json | In `traefik-acme` Docker volume | 6 LE certs via Cloudflare DNS-01 |
| Postgres | healthy | pgdata volume; all schemas migrated |
| Redis | healthy | password-protected |

**All 20 containers healthy** — 13 Go services, traefik, postgres, redis, grafana, prometheus, guest-web, admin-web.

### Live endpoints

| URL | Response | Backend |
|-----|----------|---------|
| `https://os.peoplewelike.club` | 200 | infra-guest-web-1:3000 |
| `https://www.peoplewelike.club` | 200 | infra-guest-web-1:3000 |
| `https://peoplewelike.club` | 200 | infra-guest-web-1:3000 |
| `https://admin.peoplewelike.club` | 307→/login | infra-admin-web-1:3001 |
| `https://api.peoplewelike.club` | 200 `{"status":"ok"}` | infra-gateway-1:8000 |
| `https://grafana.peoplewelike.club` | 302→/login | infra-grafana-1:3000 |

### Ingress architecture

```
Client → Cloudflare → Traefik :443 (Docker, infra_proxy network)
                                    ↓ Docker label routing
                      guest-web, admin-web, gateway, grafana
```

Traefik handles TLS via Cloudflare DNS-01. Certs in `traefik-acme` Docker volume.
**Note:** nginx was stopped at M5.6 cutover (2026-03-17). Traefik owns :80/:443 directly.

### VPS-local patches (not in GitHub main)

These fixes were applied to make M6 deploy work with Docker Compose v5 + Docker Engine 29.x:

| File | Fix |
|------|-----|
| `Dockerfile` | Alpine runtime (was distroless) — enables CMD-SHELL healthchecks |
| `infra/docker-compose.cloud.yml` | Traefik v3.6, ports 80:80/443:443, duplicate YAML `<<:` merge keys fixed, all healthchecks use `127.0.0.1` |
| `infra/traefik/traefik.yml` | Production ports :80/:443, `ping: {}`, `network: infra_proxy` |
| `web/admin/Dockerfile` | Builder `node:20-slim` (glibc for SWC); `public/` dir created |
| `web/guest/Dockerfile` | Builder `node:20-slim` (glibc for SWC) |
| `web/guest/next.config.js` | Replaced `next.config.ts` (Next.js 14 doesn't support .ts config) |
| `web/admin/src/app/(admin)/` | Removed — duplicate route group conflicting with `/admin/` structure |

Traefik handles TLS via Cloudflare DNS-01 Let's Encrypt. Certs stored in `infra/traefik/acme.json`.

---

## Radio VPS — 72.60.181.89

### What is running (stable — do not modify)

| Container | Image | Status | Port |
|-----------|-------|--------|------|
| pwl-market-app | pwl-market-pwl-market-app | healthy | 127.0.0.1:3101 |
| pwl-market-db | postgres:16-alpine | healthy | internal |
| pwl-more-app | pwl-more-pwl-more-app | healthy | 127.0.0.1:3100 |
| pwl-more-db | postgres:16-alpine | healthy | internal |
| radio-rtmp | tiangolo/nginx-rtmp | healthy | 0.0.0.0:1935 |
| radio-web | nginx:alpine | running | 0.0.0.0:8080 |
| radio-autodj | radijas-v2-autodj | running | — |
| radio-rtmp-auth | radijas-v2-rtmp-auth | running | 8088 (internal) |

### nginx vhosts on this machine

- `market.peoplewelike.club` → pwl-market-app (:3101)
- `more.peoplewelike.club` → pwl-more-app (:3100)
- `radio.peoplewelike.club` → radio-web (:8080)
- `stream.peoplewelike.club` → radio-web (:8080)

### Backup

- `pwl-market-backup.timer` — daily backup of Market Postgres, active

### Operational notes

- VPS was previously overloading due to ffmpeg compute load. Resolved by addressing compute (not SSH config).
- SSH hardening was applied here as well (same as NiteOS VPS).
- service-1 (old Fastify prototype) is NOT running here and was never deployed here.
- Do not deploy NiteOS services here. Do not modify radio or market stacks.

---

## Repository State

Last verified: 2026-03-17

| Phase | Status |
|-------|--------|
| M0 — Monorepo scaffold | Complete |
| M0 — Go workspace (15 modules) | Complete (compilation verified locally, Go 1.26.1) |
| M0 — Migrations SQL | Complete |
| M0 — Shared tooling (pkg/) | Complete |
| M0 — RS256 key pair (local dev) | Complete (in infra/secrets/, gitignored) |
| M0 — CI skeleton (.github/workflows/ci.yml) | Complete (not yet triggered — no GitHub remote) |
| M3 — All 13 Go services | Complete (build clean, critical tests pass) |
| M3 — Edge service | Complete (SQLite WAL, sync agent, catalog cache) |
| M4 — Admin console (web/admin/) | Complete (all pages, BFF pattern, iron-session) |
| M5 — infra/docker-compose.cloud.yml | Complete |
| M5 — Traefik config | Complete |
| M5 — Prometheus + Grafana stubs | Complete |
| M5 — Scripts (preflight, healthcheck, smoke-test, backup, restore) | Complete |
| M5 — Deployment docs | Complete |
| M5.6 — M5.6 cutover (Traefik on :80/:443, nginx stopped) | **Complete 2026-03-17** |
| M6 — Guest web + staff surfaces + XP backend + UID lookup + radio persistent | **Complete 2026-03-17** |

### Guest web capabilities (web/guest/)

- **Framework**: Next.js 14 App Router, TypeScript, Tailwind, iron-session BFF (24h cookie)
- **Guest surfaces**: home/landing, login, register, wallet (top-up), tickets, events, session, venues/[slug], profile
- **Staff surfaces**: `/staff/door` (checkin flow), `/staff/bar` (POS + cart), `/staff/security` (guest lookup), `/staff/manager` (live ops)
- **Persistent radio player**: global fixed-bottom HTML5 audio player in root layout; preserved across navigation
- **XP/level**: backend-authoritative from `profiles.users.global_xp` / `global_level` — not client-derived
- **NiteTap UID lookup**: `GET /profiles/users/by-nfc-uid/{uid}` wired via BFF (`/api/staff/guest-lookup-uid`)
- **Active session query**: `GET /sessions/guest/{user_id}` — registered and accessible via gateway
- **Ledger events**: `occurred_at` field used (not `created_at`)
- **NC/CHF**: 1 NC = 1 CHF enforced end-to-end

### What is NOT yet done

| Item | Blocker / Note |
|------|----------------|
| Android apps | Not started — scaffold only |
| TWINT payment provider | Waiting for Swiss business account credentials |
| Full Prometheus instrumentation | stdlib-only metrics now; `prometheus/client_golang` deferred |
| Grafana dashboard panels | Stubs only; real data requires live deployment |
| DNS cutover (os/admin.peoplewelike.club) | After M5 stack is verified healthy |
| M6 — Pilot venue provisioning | After DNS cutover |
| M7 — service-2 shared identity | Post-pilot |

### Known Technical Debt

| Item | Detail |
|------|--------|
| `security` role schema gap | Backend handlers and staff layout reference a `security` role. The `profiles.users` CHECK constraint does not include it — so the role can never be assigned. Handlers still allow it correctly (ready for when it's added). Fix: add `'security'` to the CHECK constraint in a future migration. |
| Capacity tracking | `sessions.venue_sessions` has no per-venue capacity column. `ListActiveSessions` count is the only proxy. Requires schema addition + catalog service integration. |
| Grafana dashboards | Four stub dashboards provisioned. Panels require real traffic data from a live deployment to populate meaningfully. |
| Prometheus business metrics | `pkg/metrics` is stdlib-only (no histograms, no labels). Full instrumentation requires `prometheus/client_golang` dependency. |

---

## Port Reference (authoritative)

Verified against Go source (`cmd/main.go` defaults) and `infra/docker-compose.cloud.yml`.

| Service | Port | Notes |
|---------|------|-------|
| gateway | 8000 | Public API entry point |
| auth | 8010 | Internal only |
| profiles | 8020 | Internal only |
| ledger | 8030 | Internal only |
| wallet | 8040 | Internal only |
| payments | 8050 | Internal + Stripe/TWINT webhooks via gateway |
| ticketing | 8060 | Internal only |
| devices | 8070 | Internal only |
| catalog | 8080 | Internal only |
| orders | 8090 | Internal only |
| sessions | 8100 | Internal only |
| sync | 8110 | Internal + edge connections via gateway |
| reporting | 8120 | Internal only |
| guest-web | 3000 | os.peoplewelike.club |
| admin-web | 3001 | admin.peoplewelike.club |
| Grafana | 3100 | grafana.peoplewelike.club |
| Postgres | 5432 | Internal only |
| Redis | 6379 | Internal only |
| Edge LAN API | 9000 | Venue LAN only |

---

## Key Invariants (never change without explicit review)

1. **Ledger is append-only.** No UPDATE/DELETE on `ledger.ledger_events`. Enforced at DB level (REVOKE) and app level.
2. **No NC minted without a verified payment callback.** Enforced in payments service.
3. **Gateway strips Authorization header before forwarding.** Services trust X-User-* headers from gateway only.
4. **Edge is authoritative during live ops.** Cloud is durable aggregation. These roles do not swap.
5. **Radio VPS is not touched by NiteOS deployment.** Ever.
6. **service-1 is reference material only.** It was never in production.
7. **`profiles.users.venue_id`** is set by nitecore only (`PATCH /users/{id}/venue`). Included in JWT claim.

---

## Change Log

| Date | What changed | Who |
|------|-------------|-----|
| 2026-03-17 | M6 deployed to production VPS: guest-web live at os.peoplewelike.club, admin-web with catalog/venues/items/users, XP backend, radio player, UID lookup, sessions. Fixed Docker Compose v5 YAML issues, Alpine healthchecks (127.0.0.1), SWC builds, distroless→alpine runtime, next.config.ts→js, duplicate route group removed | Claude Code |
| 2026-03-17 | Stabilization pass: port table corrected; guest-web Complete; Known Technical Debt section added | Claude Code |
| 2026-03-17 | Guest web complete: persistent radio player, XP/level backend-authoritative, NiteTap UID lookup live, session query endpoint, LedgerEvent.occurred_at fix, PROFILES_SERVICE_URL wired to orders | Claude Code |
| 2026-03-16 | Repo normalization: VPS label drift fixed, niteos.io → peoplewelike.club, DOMAIN_MODEL venue_id added, SERVICE_MAP updated to reflect actual topology, DEPLOY_NITEOS_VPS.md created, INFRA_STATE.md created | Claude Code |
| 2026-03-15 | M4 admin console complete; M5 infra artifacts complete (docker-compose.cloud.yml, scripts, Grafana stubs, traefik config) | Claude Code |
| 2026-03-15 | M3 all 13 services + edge complete; M0 hardening pass | Claude Code |
| 2026-03-15 | M0 scaffolding complete | Claude Code |
