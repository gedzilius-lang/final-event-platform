# Infrastructure State

Canonical record of what has been built, deployed, and established.
Updated as work progresses. Use this file to orient any new session or handover.

---

## Machine Inventory

| Role | IP | OS | Purpose |
|------|----|----|---------|
| **NiteOS VPS** | `31.97.126.86` | Ubuntu 22.04 LTS | NiteOS Cloud Core runtime (target, not yet deployed) |
| **Radio VPS** | `72.60.181.89` | Ubuntu 22.04 LTS | Radio + Market + More runtime (stable, do not touch) |

**VPS naming rule:** Do not use "VPS A / VPS B" labels — they were ambiguous. Use role names or IPs directly.

---

## NiteOS VPS — 31.97.126.86

### What has been established

| Item | Status | Notes |
|------|--------|-------|
| OS | Ubuntu 22.04.5 LTS | Kernel 5.15.0-164-generic |
| Docker | Installed (v29.2.1, Compose v5.0.2) | Running, enabled at boot |
| SSH hardening | Done | Key-only root login, GSSAPI/DNS disabled, fail2ban, UFW rate limit on port 22 |
| Swap | 4 GB added | Stabilizes Docker + Go runtime on low-RAM VPS |
| UFW firewall | Active | Allow: 22, 80, 443, 8080, 8443, 1935 |
| nginx (host) | Active on :80/:443 | Routes HTTPS → Traefik for NiteOS subdomains |
| nginx vhosts configured | Yes | api.peoplewelike.club, admin.peoplewelike.club, grafana.peoplewelike.club, traefik.peoplewelike.club |
| Traefik | Not running | Will run in Docker once NiteOS stack is deployed |
| /opt/niteos | ABSENT | Repo not yet cloned; stack not deployed |
| cloud.env | ABSENT | Not yet configured |
| JWT keys | ABSENT | Not yet generated on this machine |
| acme.json | ABSENT | Not yet created |
| Postgres (NiteOS) | ABSENT | No pgdata volume, no schemas |
| Redis (NiteOS) | ABSENT | Not running |
| Backup timer | ABSENT | niteos-backup.timer not installed |
| Disk free (/) | ~35 GB | Sufficient for full stack |

### What remains before first deployment

1. Push repo to GitHub (enables `git clone` from VPS)
2. Clone repo to `/opt/niteos` on NiteOS VPS
3. Copy `infra/cloud.env.example` → `infra/cloud.env`, fill in all values
4. Generate JWT keys (`infra/secrets/`)
5. Create `infra/traefik/acme.json` with `chmod 600`
6. Run `bash scripts/preflight-cloud.sh`
7. Start Postgres: `docker compose ... up -d postgres`
8. Run migrations: `bash scripts/migrate.sh`
9. Start full stack: `make cloud-up`
10. Verify: `bash scripts/healthcheck-cloud.sh`
11. Smoke test: `NITECORE_PASSWORD=<pw> bash scripts/smoke-test-pilot.sh`
12. Enable backup timer: install `infra/systemd/niteos-backup.{service,timer}`

See: `docs/DEPLOY_NITEOS_VPS.md` for the full runbook.

### Ingress architecture (already configured on this VPS)

```
Client → Cloudflare → nginx :443 (host) → proxy_pass → Traefik (Docker container)
                                                              ↓
                                              gateway, admin-web, grafana, traefik-dashboard
```

nginx vhosts route these subdomains to Traefik:
- `api.peoplewelike.club` → NiteOS gateway (:8000)
- `admin.peoplewelike.club` → admin-web (:3001)
- `grafana.peoplewelike.club` → Grafana (:3100)
- `traefik.peoplewelike.club` → Traefik dashboard

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

Last verified: 2026-03-16

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

### What is NOT yet done

| Item | Blocker / Note |
|------|----------------|
| GitHub remote + CI running | Repo not yet pushed to GitHub |
| NiteOS VPS deployment | Blocked on GitHub push (for git clone) |
| Guest web (web/guest/) | Not started — not required for admin-first pilot |
| Android apps | Not started — scaffold only |
| TWINT payment provider | Waiting for Swiss business account credentials |
| Full Prometheus instrumentation | stdlib-only metrics now; `prometheus/client_golang` deferred |
| Grafana dashboard panels | Stubs only; real data requires live deployment |
| DNS cutover (os/admin.peoplewelike.club) | After M5 stack is verified healthy |
| M6 — Pilot venue provisioning | After DNS cutover |
| M7 — service-2 shared identity | Post-pilot |

---

## Port Reference (authoritative)

| Service | Port | Notes |
|---------|------|-------|
| gateway | 8000 | Public API entry point |
| auth | 8010 | Internal only |
| profiles | 8020 | Internal only |
| ledger | 8030 | Internal only |
| wallet | 8040 | Internal only |
| payments | 8050 | Internal + Stripe/TWINT webhooks via gateway |
| ticketing | 8060 | Internal only |
| orders | 8070 | Internal only |
| catalog | 8080 | Internal only |
| devices | 8090 | Internal only |
| sessions | 8100 | Internal only |
| reporting | 8110 | Internal only |
| sync | 8120 | Internal + edge connections via gateway |
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
| 2026-03-16 | Repo normalization: VPS label drift fixed, niteos.io → peoplewelike.club, DOMAIN_MODEL venue_id added, SERVICE_MAP updated to reflect actual topology, DEPLOY_NITEOS_VPS.md created, INFRA_STATE.md created | Claude Code |
| 2026-03-15 | M4 admin console complete; M5 infra artifacts complete (docker-compose.cloud.yml, scripts, Grafana stubs, traefik config) | Claude Code |
| 2026-03-15 | M3 all 13 services + edge complete; M0 hardening pass | Claude Code |
| 2026-03-15 | M0 scaffolding complete | Claude Code |
