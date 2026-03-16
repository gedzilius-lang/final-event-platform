# VPS A: Rebuild vs. In-Place Overwrite Decision Framework

Complete `docs/VPS_A_AUDIT_CHECKLIST.md` first. Then apply the rules below.

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
- VPS has been provisioned or reinstalled with Ubuntu 24.04 LTS
- You have SSH root access
- DNS A record for `niteos.io` → VPS IP is already set in Cloudflare

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

## §4 — Co-existence with service-1 (Port Conflict)

If service-1 is running nginx on 80/443 and you are not yet ready to cut over:

```bash
# Deploy NiteOS on offset ports first (§14 of DEPLOY_VPS_A.md)
# 1. Edit entrypoints in infra/traefik/traefik.yml:
#    web:  address: ":8080"
#    websecure: address: ":8443"
# 2. Open firewall ports 8080, 8443
ufw allow 8080/tcp
ufw allow 8443/tcp
# 3. Add a staging subdomain (staging.niteos.io → VPS IP) in Cloudflare
# 4. Deploy and verify at https://staging.niteos.io:8443
# 5. When ready to cut over, stop nginx, restore ports 80/443 in traefik.yml, restart Traefik.
```

**Cutover sequence:**
```bash
# At cutover window (low traffic):
systemctl stop nginx
# or: docker stop <nginx-container>

# Restore traefik.yml to :80/:443
cd /opt/niteos
# edit infra/traefik/traefik.yml — set web: ":80" and websecure: ":443"
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  restart traefik

# Confirm TLS
curl -I https://api.niteos.io/healthz
```

---

## Quick-Reference Summary

| Scenario | Path |
|----------|------|
| Fresh VPS, nothing installed | PATH A |
| OS reinstalled | PATH A |
| Existing partial deploy, no data worth keeping | PATH A |
| Existing full deploy, live data, no port conflict | PATH B |
| Existing deploy, port conflict with service-1 | §4 co-existence, then PATH B cutover |
| Unknown/compromised state | PATH A (full wipe) |
