# NiteOS VPS Deployment Runbook

> **This file is also saved as `docs/DEPLOY_NITEOS_VPS.md`.** Both names are kept during transition. The canonical name going forward is `DEPLOY_NITEOS_VPS.md`.

**Target:** NiteOS VPS — 31.97.126.86 (Ubuntu 22.04 LTS, Docker 29.2.1, SSH hardened)
**Domain:** peoplewelike.club subdomains — Cloudflare DNS-01 TLS via Traefik
**Ingress:** nginx (host) routes :443 → Traefik in Docker
**Services:** 13 Go microservices + Postgres + Redis + Traefik + Admin Web + Guest Web + Prometheus + Grafana

---

## 0. Prerequisites (before you touch the VPS)

| Item | Where |
|------|-------|
| Cloudflare API token (Zone:DNS:Edit + Zone:Zone:Read) | Cloudflare dashboard → Profile → API Tokens |
| Stripe live API key + webhook secret | Stripe dashboard |
| GitHub SSH deploy key | `ssh-keygen -t ed25519 -C deploy@vps-a` → add public key to repo |
| Domain DNS A record pointing to VPS IP | Cloudflare → DNS |

---

## 1. Provision VPS

Recommended spec: **4 vCPU · 8 GB RAM · 80 GB SSD · Ubuntu 24.04 LTS**

```bash
# Update system
apt-get update && apt-get upgrade -y

# Install Docker (official script)
curl -fsSL https://get.docker.com | sh
systemctl enable --now docker

# Verify
docker --version
docker compose version   # must be ≥ 2.24
```

---

## 2. Clone repository

```bash
mkdir -p /opt/niteos
cd /opt/niteos

# Copy deploy key first if using SSH:
# install -m 600 /path/to/deploy_key ~/.ssh/id_ed25519_deploy
# export GIT_SSH_COMMAND="ssh -i ~/.ssh/id_ed25519_deploy"

git clone git@github.com:ORG/final-event-platform.git .
```

---

## 3. Configure environment

```bash
cd /opt/niteos

# Copy template; edit ALL values before proceeding
cp infra/cloud.env.example infra/cloud.env
chmod 600 infra/cloud.env
nano infra/cloud.env
```

**Required values to fill in `infra/cloud.env`:**

| Variable | How to generate |
|----------|----------------|
| `POSTGRES_PASSWORD` | `openssl rand -base64 32` |
| `REDIS_PASSWORD` | `openssl rand -base64 32` |
| `SESSION_SECRET` | `openssl rand -base64 32` (admin console cookie) |
| `GUEST_SESSION_SECRET` | `openssl rand -base64 32` (guest web cookie — separate secret) |
| `GRAFANA_ADMIN_PASSWORD` | `openssl rand -base64 32` |
| `CF_DNS_API_TOKEN` | Cloudflare API token (see §0) |
| `TRAEFIK_DASHBOARD_USERS` | `htpasswd -nB admin` (install: `apt-get install apache2-utils`) |
| `STRIPE_API_KEY` | Stripe live key (`sk_live_...`) |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook secret (`whsec_...`) |
| `STRIPE_PUBLISHABLE_KEY` | Stripe publishable key (`pk_live_...`) — safe for browser |
| `DOMAIN` | `peoplewelike.club` |
| `ACME_EMAIL` | `admin@peoplewelike.club` |

Run the preflight check to confirm all values are set:
```bash
bash scripts/preflight-cloud.sh
```

---

## 4. Generate JWT keys

```bash
mkdir -p infra/secrets
chmod 700 infra/secrets

openssl genrsa -out infra/secrets/jwt_private_key.pem 2048
openssl rsa -in infra/secrets/jwt_private_key.pem -pubout \
  -out infra/secrets/jwt_public_key.pem
chmod 600 infra/secrets/jwt_private_key.pem
```

---

## 5. Prepare Traefik TLS storage

```bash
touch infra/traefik/acme.json
chmod 600 infra/traefik/acme.json
```

---

## 6. Run database migrations

Postgres must be healthy before migrations. Start only the DB:

```bash
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  up -d postgres

# Wait ~10 s for postgres to be ready, then:
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  exec postgres pg_isready -U niteos -d niteos

# Run migrations (psql must be on PATH or use a temp container)
# Option A — psql on host:
export DATABASE_URL="postgres://niteos:$(grep POSTGRES_PASSWORD infra/cloud.env | cut -d= -f2)@localhost:5432/niteos?sslmode=disable"
bash scripts/migrate.sh

# Option B — via docker exec:
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  exec postgres psql -U niteos -d niteos -c "\dn"   # verify schemas exist
```

---

## 7. Start the full stack

```bash
make cloud-up
# or: docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env up -d

# Watch startup
make cloud-logs   # Ctrl+C to stop tailing
make cloud-ps     # all services should be "healthy" within ~60s
```

---

## 8. Verify deployment

```bash
# Built-in health check (hits all /healthz endpoints)
bash scripts/healthcheck-cloud.sh

# Smoke test (creates and exercises pilot flows)
bash scripts/smoke-test-pilot.sh
```

Manual checks:

| URL | Expected |
|-----|---------|
| `https://api.peoplewelike.club/healthz` | `{"status":"ok"}` |
| `https://os.peoplewelike.club` | Guest web landing page |
| `https://admin.peoplewelike.club` | Admin console login |
| `https://grafana.peoplewelike.club` | Grafana login |
| `https://traefik.peoplewelike.club` | Traefik dashboard (HTTP Basic Auth) |

---

## 9. First pilot venue setup

After the stack is healthy, create the pilot venue and admin user:

```bash
# 1. Register the venue_admin user (via admin console or curl):
curl -s -X POST https://api.peoplewelike.club/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@venue-name.com","password":"STRONG_PASSWORD","display_name":"Venue Admin"}'
# Returns: {"user_id":"...","access_token":"..."}

# 2. Store the nitecore token (login as nitecore):
NITECORE_TOKEN=$(curl -s -X POST https://api.peoplewelike.club/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"nitecore@peoplewelike.club","password":"NITECORE_PASSWORD"}' \
  | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

# 3. Create venue in catalog:
VENUE_ID=$(curl -s -X POST https://api.peoplewelike.club/catalog/venues \
  -H "Authorization: Bearer $NITECORE_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Venue Name","slug":"venue-name","city":"Zurich","capacity":200,"staff_pin":"1234","timezone":"Europe/Zurich"}' \
  | grep -o '"venue_id":"[^"]*"' | cut -d'"' -f4)
echo "Venue ID: $VENUE_ID"

# 4. Assign venue_admin role + venue_id to the user:
USER_ID="<user_id from step 1>"
curl -s -X PATCH https://api.peoplewelike.club/profiles/users/$USER_ID/venue \
  -H "Authorization: Bearer $NITECORE_TOKEN" \
  -H 'Content-Type: application/json' \
  -d "{\"venue_id\":\"$VENUE_ID\",\"role\":\"venue_admin\"}"

# 5. Login to admin console at https://admin.peoplewelike.club with venue_admin credentials
```

---

## 10. Enable backup timer

```bash
# Copy env template, fill in DATABASE_URL and upload settings
cp infra/backup.env.example /opt/niteos/backup.env
chmod 600 /opt/niteos/backup.env
nano /opt/niteos/backup.env

# Install systemd units
cp infra/systemd/niteos-backup.service /etc/systemd/system/
cp infra/systemd/niteos-backup.timer   /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now niteos-backup.timer

# Verify
systemctl status niteos-backup.timer
systemctl list-timers niteos-backup.timer

# Run a manual backup to confirm:
systemctl start niteos-backup.service
journalctl -u niteos-backup.service --no-pager
```

---

## 11. Rollback procedure

```bash
# Stop stack, revert to previous image tag, restart
make cloud-down
git log --oneline -5        # find good commit
git checkout <commit>
make cloud-build
make cloud-up
bash scripts/healthcheck-cloud.sh
```

---

## 12. Restore from backup

```bash
make cloud-down
export DATABASE_URL="postgres://niteos:<POSTGRES_PASSWORD>@localhost:5432/niteos?sslmode=disable"
bash scripts/restore.sh /opt/niteos/backups/niteos_YYYYMMDD_HHMMSS.sql.gz
make cloud-up
bash scripts/healthcheck-cloud.sh
```

---

## 13. Ongoing operations

| Task | Command |
|------|---------|
| View logs | `make cloud-logs` |
| Restart single service | `docker compose -f infra/docker-compose.cloud.yml restart <service>` |
| Pull new code + redeploy | `git pull && make cloud-build && make cloud-up` |
| Run migrations | `make cloud-migrate` |
| View service status | `make cloud-ps` |
| Manual backup | `systemctl start niteos-backup.service` |
| Check Prometheus targets | `https://grafana.peoplewelike.club` → Explore → Prometheus |

---

## 14. nginx → Traefik ingress setup

The NiteOS VPS uses nginx (host) as the public ingress on :443, routing to Traefik in Docker.
Add an nginx `server` block for each NiteOS subdomain. Example for all NiteOS subdomains:

```nginx
# /etc/nginx/sites-available/niteos-proxy.conf
# Add a block for each subdomain — all proxy_pass to Traefik's websecure port.

upstream traefik_backend {
    server 127.0.0.1:8443;
}

# api.peoplewelike.club
server {
    listen 443 ssl;
    server_name api.peoplewelike.club;
    ssl_certificate     /etc/letsencrypt/live/peoplewelike.club/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/peoplewelike.club/privkey.pem;
    location / {
        proxy_pass https://traefik_backend;
        proxy_ssl_verify off;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# Repeat the server block for:
#   admin.peoplewelike.club
#   os.peoplewelike.club
#   grafana.peoplewelike.club
#   traefik.peoplewelike.club
#
# Also add blocks for apex domain + www:
#   peoplewelike.club      → os.peoplewelike.club (redirect or proxy)
#   www.peoplewelike.club  → os.peoplewelike.club (redirect or proxy)
```

Enable and reload:
```bash
sudo ln -sf /etc/nginx/sites-available/niteos-proxy.conf /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

If for any reason Traefik needs to bind directly to :80/:443 (without nginx in front):
1. Edit `infra/traefik/traefik.yml` — set entryPoints web to `:80` and websecure to `:443`
2. Update compose Traefik ports to `"80:80"` and `"443:443"`
3. Remove the nginx proxy_pass blocks above

For the standard nginx→Traefik setup (recommended), no changes to traefik.yml are needed.

---

## Troubleshooting

| Symptom | Check |
|---------|-------|
| Services unhealthy | `docker compose ... logs <service>` — look for DB connection errors |
| TLS cert not issuing | Check `CF_DNS_API_TOKEN` perms; `docker compose ... logs traefik` for ACME errors |
| Auth 401 on valid token | JWT key mismatch — confirm `infra/secrets/jwt_private_key.pem` is the one used at registration |
| Postgres not ready | `docker compose ... exec postgres pg_isready -U niteos` |
| Migrations failed | Check `scripts/migrate.sh` output; re-run is idempotent for `CREATE TABLE IF NOT EXISTS` |
