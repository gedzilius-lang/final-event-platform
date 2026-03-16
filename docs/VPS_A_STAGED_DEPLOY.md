# VPS A: Staged Coexistence Deployment

**Situation:** nginx is live on VPS A ports 80/443 serving:
- `market.peoplewelike.club`
- `more.peoplewelike.club`
- `radio.peoplewelike.club`
- `stream.peoplewelike.club`

These domains must remain unaffected. NiteOS deploys alongside them on offset ports
until nginx is intentionally decommissioned.

---

## Port Allocation

| Service | Ports | Owner |
|---------|-------|-------|
| HTTP | `:80` | nginx (do not touch) |
| HTTPS | `:443` | nginx (do not touch) |
| nginx internal | `127.0.0.1:8088`, `127.0.0.1:8089` | nginx (do not touch) |
| NiteOS HTTP → HTTPS redirect | `:8080` | Traefik (new) |
| NiteOS HTTPS | `:8443` | Traefik (new) |

**8080 and 8443 are confirmed free** from the VPS A audit.

---

## What Was Changed in the Repo

These changes are already committed:

| File | Change |
|------|--------|
| `infra/traefik/traefik.yml` | `web: :8080`, `websecure: :8443` (was 80/443) |
| `infra/docker-compose.cloud.yml` | Traefik ports `8080:8080` / `8443:8443` |
| `infra/docker-compose.cloud.yml` | `GATEWAY_URL`, `ADMIN_ORIGIN`, `GF_SERVER_ROOT_URL` use `${TRAEFIK_PORT_SUFFIX}` |
| `infra/cloud.env.example` | `TRAEFIK_PORT_SUFFIX=:8443` (staging) / empty (production) |
| `scripts/preflight-cloud.sh` | Validates `TRAEFIK_PORT_SUFFIX`; accepts empty value at production |

---

## TLS Note

TLS still works on `:8443`. Traefik uses **Cloudflare DNS-01** for Let's Encrypt — the
challenge is resolved via DNS API, not HTTP. Port 80 is not required for cert issuance.
Certs for `api.niteos.io`, `admin.niteos.io`, `grafana.niteos.io`, `traefik.niteos.io`
will be issued correctly on first request.

---

## Staged Deploy: Step-by-Step

### Pre-conditions (complete before starting)

- [ ] Cloudflare API token created with Zone:DNS:Edit + Zone:Zone:Read for `niteos.io`
- [ ] DNS A records added in Cloudflare for VPS A IP (proxied = OFF for staging; or DNS-only):
  - `api.niteos.io` → VPS A public IP
  - `admin.niteos.io` → VPS A public IP
  - `grafana.niteos.io` → VPS A public IP
  - `traefik.niteos.io` → VPS A public IP
- [ ] Stripe webhook URL noted as `https://api.niteos.io:8443/payments/webhook` in Stripe dashboard
- [ ] Firewall allows 8080 and 8443 TCP

### Step 1: Open firewall ports (run on VPS A as root)

```bash
ufw allow 8080/tcp comment "NiteOS Traefik HTTP staging"
ufw allow 8443/tcp comment "NiteOS Traefik HTTPS staging"
ufw status numbered
# Confirm 8080 and 8443 appear; 80 and 443 still belong to nginx
```

### Step 2: Clone repository

```bash
mkdir -p /opt/niteos
cd /opt/niteos
git clone https://github.com/ORG/final-event-platform.git .
# Use SSH deploy key if the repo is private:
# install -m 600 /path/to/deploy_key ~/.ssh/id_ed25519_deploy
# GIT_SSH_COMMAND="ssh -i ~/.ssh/id_ed25519_deploy -o StrictHostKeyChecking=no" \
#   git clone git@github.com:ORG/final-event-platform.git .
```

### Step 3: Configure environment

```bash
cd /opt/niteos
cp infra/cloud.env.example infra/cloud.env
chmod 600 infra/cloud.env
nano infra/cloud.env
```

Fill in every value. Key staging-specific values:

```bash
DOMAIN=niteos.io
TRAEFIK_PORT_SUFFIX=:8443           # IMPORTANT: colon-prefix included
STRIPE_API_KEY=sk_test_...          # use test key until cutover to production traffic
STRIPE_WEBHOOK_SECRET=whsec_...     # must match Stripe webhook endpoint for :8443
```

All other secrets: generate with `openssl rand -base64 32`.

### Step 4: Generate JWT keys

```bash
mkdir -p infra/secrets
chmod 700 infra/secrets
openssl genrsa -out infra/secrets/jwt_private_key.pem 2048
openssl rsa -in infra/secrets/jwt_private_key.pem -pubout \
  -out infra/secrets/jwt_public_key.pem
chmod 600 infra/secrets/jwt_private_key.pem
```

### Step 5: Prepare Traefik TLS storage

```bash
touch infra/traefik/acme.json
chmod 600 infra/traefik/acme.json
```

### Step 6: Run preflight

```bash
bash scripts/preflight-cloud.sh
```

All items must be green. The `TRAEFIK_PORT_SUFFIX` check will show "set" (`:8443`). Fix
any failures before continuing.

### Step 7: Start Postgres, run migrations

```bash
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  up -d postgres

sleep 15

docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  exec postgres pg_isready -U niteos -d niteos

export DATABASE_URL="postgres://niteos:$(grep POSTGRES_PASSWORD infra/cloud.env | cut -d= -f2)@localhost:5432/niteos?sslmode=disable"
bash scripts/migrate.sh
```

### Step 8: Start full stack

```bash
make cloud-up
# or:
# docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env up -d

# Watch startup (Ctrl+C to stop tailing):
make cloud-logs

# All services should reach "healthy" within 90s:
make cloud-ps
```

### Step 9: Verify

```bash
# Internal health check via docker exec (always works):
bash scripts/healthcheck-cloud.sh

# Public endpoint checks (requires DNS to have propagated):
curl -sk https://api.niteos.io:8443/healthz
# Expected: {"status":"ok"}

curl -sk -o /dev/null -w "%{http_code}" https://admin.niteos.io:8443/api/health
# Expected: 200

# Smoke test (creates test venue + guest):
NITECORE_PASSWORD=<your-nitecore-password> \
  API_BASE=https://api.niteos.io:8443 \
  bash scripts/smoke-test-pilot.sh
```

### Step 10: Enable backup timer

```bash
cp infra/backup.env.example /opt/niteos/backup.env
chmod 600 /opt/niteos/backup.env
nano /opt/niteos/backup.env   # fill in DATABASE_URL and optional upload target

cp infra/systemd/niteos-backup.service /etc/systemd/system/
cp infra/systemd/niteos-backup.timer   /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now niteos-backup.timer
systemctl start niteos-backup.service    # test run
journalctl -u niteos-backup.service --no-pager
```

---

## Access During Staged Deployment

| URL | Service |
|-----|---------|
| `https://api.niteos.io:8443` | Gateway API |
| `https://admin.niteos.io:8443` | Admin console |
| `https://grafana.niteos.io:8443` | Grafana |
| `https://traefik.niteos.io:8443` | Traefik dashboard (HTTP Basic Auth) |
| `https://api.niteos.io:8443/healthz` | Stack health |

All `*.peoplewelike.club` domains continue to be served normally by nginx on `:443`.

---

## Edge Device Config During Staged Deployment

Any NiteOS Edge device (Master Tablet) must point its cloud sync URL to the offset port:

```
CLOUD_SYNC_URL=https://api.niteos.io:8443
```

This is set in the edge device's local config, not in this repo.

---

## Nginx Config Validation (run after NiteOS deploy, before cutover)

Confirm nginx is still healthy and your domains are unaffected:

```bash
nginx -t
curl -sI https://market.peoplewelike.club | head -3
curl -sI https://more.peoplewelike.club | head -3
curl -sI https://radio.peoplewelike.club | head -3
curl -sI https://stream.peoplewelike.club | head -3
```

All should return HTTP 200 or the expected response. If any fail, NiteOS has not
affected them (different ports), so investigate nginx independently.

---

## Cutover to Port 80/443 (future — when nginx is decommissioned)

**Prerequisites before cutover:**
- NiteOS has been running stably on 8080/8443 for a pilot period
- All `*.peoplewelike.club` domains are confirmed migrated or decommissioned
- Backup taken: `systemctl start niteos-backup.service`

**Cutover sequence** (takes ~2 minutes; brief TLS re-negotiation on port change):

```bash
cd /opt/niteos

# 1. Stop nginx (or the specific vhosts if nginx will continue on other ports)
systemctl stop nginx
# Confirm ports 80 and 443 are free:
ss -tlnp | grep -E ':80\b|:443\b'

# 2. Update Traefik entrypoints
nano infra/traefik/traefik.yml
# Change:
#   web:       address: ":8080"  →  address: ":80"
#   websecure: address: ":8443"  →  address: ":443"

# 3. Update compose port bindings
nano infra/docker-compose.cloud.yml
# Change under traefik service:
#   - "8080:8080"  →  - "80:80"
#   - "8443:8443"  →  - "443:443"

# 4. Update port suffix in cloud.env
nano infra/cloud.env
# Change:
#   TRAEFIK_PORT_SUFFIX=:8443  →  TRAEFIK_PORT_SUFFIX=

# 5. Update firewall
ufw allow 80/tcp
ufw allow 443/tcp
ufw delete allow 8080/tcp
ufw delete allow 8443/tcp

# 6. Rolling restart (Traefik and admin-web only; data services unaffected)
make cloud-down
make cloud-up

# 7. Verify
bash scripts/healthcheck-cloud.sh
curl -s https://api.niteos.io/healthz     # standard port, no :8443
NITECORE_PASSWORD=<pw> bash scripts/smoke-test-pilot.sh
```

**After cutover:**
- Update Stripe webhook URL from `:8443` to standard HTTPS (no port)
- Update any edge device `CLOUD_SYNC_URL` to remove `:8443`
- Update `TRAEFIK_PORT_SUFFIX=` in `cloud.env.example` comment to reflect production state
