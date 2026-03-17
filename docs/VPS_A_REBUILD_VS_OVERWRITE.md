# NiteOS VPS (31.97.126.86): Rebuild vs. In-Place Overwrite Decision Framework

> **Machine:** NiteOS VPS — 31.97.126.86. This framework does NOT apply to the Radio VPS (72.60.181.89).
>
> **Current state (2026-03-16):** /opt/niteos does not exist. First deploy → always use PATH A.
> The service-1 co-existence scenario (§4) is no longer relevant — service-1 was never deployed
> to the NiteOS VPS and does not need to be considered.

Complete `docs/VPS_A_AUDIT_CHECKLIST.md` first (run against 31.97.126.86). Then apply the rules below.

---

## Decision Tree

```
Does /opt/niteos exist with a live stack?
├── NO  → PATH A (clean deploy — see §2)
└── YES
    ├── Is the pgdata volume present and populated?
    │   └── YES → take backup first (docs/VPS_A_BACKUP_AND_ROLLBACK.md §1)
    ├── Is the git repo at the correct commit?
    ├── Are cloud.env + JWT keys + acme.json present and valid?
    └── Is service-1 running on port 80 or 443?
        ├── NO conflict → PATH B (in-place — see §3)
        └── YES conflict → read §4 (co-existence / port-offset staging)
```

---

## Mandatory Wipe Criteria (force PATH A)

Wipe and rebuild if ANY of the following are true:

| Condition | Why |
|-----------|-----|
| OS is not Ubuntu 22.04 or 24.04 | Traefik/Docker requirements; unknown system state |
| Docker version < 23 | Compose v2 plugin unavailable or broken |
| Disk has < 15 GB free after Docker install | Stack + images + pgdata needs headroom |
| `/opt/niteos` exists but no `infra/cloud.env` and no pgdata volume | Half-deployed state with no data worth preserving |
| Unknown processes bound to 80/443 that cannot be identified or stopped safely | Port conflict will block Traefik |
| System has been compromised or is in unknown state | Security |

---

## PATH A — Clean Rebuild on Fresh Ubuntu 24.04

Use when: fresh VPS, OS reinstalled, or wipe criteria above are met.

### Pre-conditions
- NiteOS VPS (31.97.126.86) has Ubuntu 22.04 LTS, Docker 29.2.1, SSH hardened (already done)
- You have SSH root access
- Cloudflare DNS A records point to 31.97.126.86 for api/admin/grafana/traefik.peoplewelike.club

### Steps

```bash
# ── Step 1: System update ─────────────────────────────────────────────────────
apt-get update && apt-get upgrade -y
apt-get install -y git curl wget nano htop ufw apache2-utils

# ── Step 2: Firewall ──────────────────────────────────────────────────────────
ufw allow 22/tcp    # SSH
ufw allow 80/tcp    # HTTP (Traefik → redirect to HTTPS)
ufw allow 443/tcp   # HTTPS
ufw --force enable

# ── Step 3: Docker (official script) ─────────────────────────────────────────
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker
docker --version         # must print version
docker compose version   # must be ≥ 2.24

# ── Step 4: Create deploy user (optional but recommended) ────────────────────
# If deploying as root, skip this block.
useradd -m -s /bin/bash niteos
usermod -aG docker niteos
# Then continue as niteos user or stay as root.

# ── Step 5: Clone repo ────────────────────────────────────────────────────────
mkdir -p /opt/niteos
cd /opt/niteos
# (Add GitHub deploy key to ~/.ssh/ first if using SSH)
git clone https://github.com/ORG/final-event-platform.git .
# or:
# git clone git@github.com:ORG/final-event-platform.git .

# ── Step 6–14: Follow docs/DEPLOY_VPS_A.md exactly from §3 onwards ───────────
#   §3  Configure infra/cloud.env
#   §4  Generate JWT keys
#   §5  Prepare infra/traefik/acme.json
#   §6  Run database migrations
#   §7  Start full stack (make cloud-up)
#   §8  Verify (healthcheck + smoke test)
#   §9  First pilot venue setup
#   §10 Enable backup timer
```

**Estimated wall time:** 20–35 minutes on a clean VPS with good network.

---

## PATH B — Safe In-Place Overwrite

Use when: `/opt/niteos` already exists with valid secrets + live pgdata, and there is no blocking port conflict.

**Rule:** never overwrite secrets (`cloud.env`, JWT keys, `acme.json`) unless you explicitly intend to rotate them. Pull new code, rebuild images, restart stack.

### Pre-conditions
- Backup has been taken and verified (see `docs/VPS_A_BACKUP_AND_ROLLBACK.md §1`)
- `cloud.env`, JWT keys, and `acme.json` are present and valid
- `make cloud-preflight` passes

### Steps

```bash
cd /opt/niteos

# ── Step 1: Confirm current state ─────────────────────────────────────────────
git log --oneline -3
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env ps

# ── Step 2: Pull new code ─────────────────────────────────────────────────────
git fetch origin
git status    # confirm no local changes that shouldn't be overwritten

# If there are local changes that need to be preserved (e.g., edited cloud.env
# is tracked — it shouldn't be, but check):
git stash    # only if needed

git pull origin main

# ── Step 3: Run preflight ─────────────────────────────────────────────────────
bash scripts/preflight-cloud.sh
# Fix any failures before continuing.

# ── Step 4: Rebuild images ────────────────────────────────────────────────────
make cloud-build

# ── Step 5: Apply any new migrations ─────────────────────────────────────────
# (safe to run even if no new migrations — scripts/migrate.sh uses IF NOT EXISTS)
make cloud-migrate

# ── Step 6: Rolling restart ───────────────────────────────────────────────────
# Option A — full stack restart (< 30 s downtime):
make cloud-down
make cloud-up

# Option B — service-by-service restart (near-zero downtime, more complex):
# docker compose ... up -d --no-deps --build <service>
# Recommended only once health checks are automated.

# ── Step 7: Verify ────────────────────────────────────────────────────────────
bash scripts/healthcheck-cloud.sh
bash scripts/smoke-test-pilot.sh
```

### What NOT to touch during in-place overwrite

| File/Directory | Why |
|----------------|-----|
| `infra/cloud.env` | All service credentials — rotation requires coordinated restart |
| `infra/secrets/jwt_private_key.pem` | Rotations invalidate all live tokens |
| `infra/traefik/acme.json` | Contains issued TLS certs — deletion forces re-issue (rate limited) |
| `pgdata` Docker volume | Live database — never delete without confirmed backup |

---

## §4 — Co-existence with service-1 (Port Conflict) — SUPERSEDED

> **This section is no longer applicable.**
> service-1 was never deployed to the NiteOS VPS (31.97.126.86).
> The NiteOS VPS has nginx routing :443 → Traefik; no port conflict exists.
> No staged coexistence steps are needed for first deployment — go directly to PATH A.

If a future scenario arises where another process holds :80/:443 on the NiteOS VPS,
refer to `docs/VPS_A_STAGED_DEPLOY.md` for the port-offset approach pattern (even though
that doc was written for a different scenario, the port-offset technique is reusable).

---

## Quick-Reference Summary

| Scenario | Path |
|----------|------|
| First deploy — /opt/niteos does not exist (current state) | PATH A |
| OS reinstalled | PATH A |
| Existing partial deploy, no data worth keeping | PATH A |
| Existing full deploy, live data, no port conflict | PATH B |
| Unknown/compromised state | PATH A (full wipe) |
