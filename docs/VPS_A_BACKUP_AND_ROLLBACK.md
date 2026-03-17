# NiteOS VPS (31.97.126.86): Pre-Deploy Backup and Rollback

> **Machine:** NiteOS VPS — 31.97.126.86. Not applicable to the Radio VPS (72.60.181.89).
>
> **Current state (2026-03-16):** No NiteOS stack deployed yet. §1 (pre-deploy backup) does not
> apply until after the first successful deployment. §2 (service-1 backup) is permanently irrelevant —
> service-1 was never on this machine. §5 (rollback to service-1) is permanently irrelevant.

Run §1 before any PATH B in-place overwrite.
Run §3 or §4 to recover from a failed deploy.

---

## §1 — Pre-Deploy Backup Checklist

Complete all items before touching the running stack.

### 1.1 Database backup

```bash
cd /opt/niteos

# Create backup directory if not present
mkdir -p /opt/niteos/backups

# Dump Postgres via Docker (does not require host psql)
PGPASSWORD=$(grep POSTGRES_PASSWORD infra/cloud.env | cut -d= -f2) \
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  exec -T postgres \
  pg_dump -U niteos -d niteos --no-password \
  | gzip > /opt/niteos/backups/pre_deploy_$(date +%Y%m%d_%H%M%S).sql.gz

# Confirm backup file is non-zero
ls -lh /opt/niteos/backups/pre_deploy_*.sql.gz | tail -1
```

Expected: file size > 20 KB for a schema-only database; larger if data exists.

### 1.2 Verify backup is readable

```bash
BACKUP_FILE=$(ls -t /opt/niteos/backups/pre_deploy_*.sql.gz | head -1)
zcat "$BACKUP_FILE" | head -20    # should print SQL header and SET commands
```

### 1.3 Snapshot secrets (copy to a safe location off-VPS)

```bash
# NEVER commit these files to git. Copy to a secure off-VPS location.
# Options: local encrypted USB, Bitwarden Secure Note, S3 with SSE-KMS.
cat infra/secrets/jwt_private_key.pem    # copy and store securely
cat infra/secrets/jwt_public_key.pem
cat infra/cloud.env                       # contains all credentials

# Confirm permissions
stat -c "%a %n" infra/secrets/jwt_private_key.pem infra/cloud.env
# Expected: 600 for both
```

### 1.4 Record current Docker state

```bash
# Save current image digests so you can return to exact state
docker images --format "{{.Repository}}:{{.Tag}} {{.ID}}" > /opt/niteos/backups/image_list_$(date +%Y%m%d_%H%M%S).txt
cat /opt/niteos/backups/image_list_*.txt | tail -20

# Save current git commit
git rev-parse HEAD > /opt/niteos/backups/git_head_$(date +%Y%m%d_%H%M%S).txt
git log --oneline -5
```

### 1.5 Pre-deploy checklist sign-off

| Item | Status |
|------|--------|
| Database backup file present and non-zero | [ ] |
| Backup file passes `zcat | head` readability check | [ ] |
| Secrets copied to secure off-VPS location | [ ] |
| Current git commit noted | [ ] |
| Docker image list saved | [ ] |
| `make cloud-preflight` passes on new code | [ ] |

**Do not proceed to PATH B until all items are checked.**

---

## §2 — service-1 Backup — IRRELEVANT

> service-1 was never deployed to the NiteOS VPS (31.97.126.86). This section does not apply
> and can be ignored entirely.

If service-1 is a running application with its own data, back it up before port cutover:

```bash
# Identify service-1's data
ls /opt/service-1/ 2>/dev/null
# Check for its own database connections
cat /opt/service-1/.env 2>/dev/null || cat /opt/service-1/config/* 2>/dev/null | grep -i "database\|postgres\|redis"

# If service-1 has its own Postgres database, dump it:
# (adjust DB name, user, and port to match service-1 config)
pg_dump -U <service1_user> -d <service1_db> | gzip > /opt/service-1/backups/pre_cutover_$(date +%Y%m%d_%H%M%S).sql.gz

# Stop service-1 only after NiteOS is verified healthy:
# systemctl stop service-1   (or: docker stop <service-1-container>)
```

---

## §3 — Rollback: Failed In-Place Update (PATH B failure)

Use when: `make cloud-up` succeeded but health checks fail and you need to revert to the previous working state.

### If code was updated but stack is still running on old images

```bash
cd /opt/niteos

# Stop new stack
make cloud-down

# Revert to previous commit
git log --oneline -5    # identify last known-good commit
git checkout <commit>   # e.g.: git checkout abc1234

# Rebuild from old code
make cloud-build

# Restart
make cloud-up

# Verify
bash scripts/healthcheck-cloud.sh
```

### If stack is down and postgres data is intact

```bash
cd /opt/niteos

# Revert code
git checkout <last-good-commit>
make cloud-build

# Start just postgres first to confirm data
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env up -d postgres
sleep 10
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  exec postgres psql -U niteos -d niteos -c "\dn"

# If schemas are present, start full stack
make cloud-up
bash scripts/healthcheck-cloud.sh
```

---

## §4 — Restore from Backup

Use when: pgdata volume is corrupt or missing, or a migration corrupted data.

```bash
cd /opt/niteos

# Stop all containers (Postgres must be stopped)
make cloud-down

# Start only Postgres with a fresh volume
# WARNING: this destroys the existing pgdata volume
docker volume rm niteos_pgdata 2>/dev/null || true
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env up -d postgres
sleep 15

# Confirm Postgres is ready
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  exec postgres pg_isready -U niteos

# Restore from backup file
BACKUP_FILE="/opt/niteos/backups/<your-backup-file>.sql.gz"
PGPASSWORD=$(grep POSTGRES_PASSWORD infra/cloud.env | cut -d= -f2)

zcat "$BACKUP_FILE" | \
  docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  exec -T postgres psql -U niteos -d niteos

# Verify schemas restored
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  exec postgres psql -U niteos -d niteos -c "\dn"

# Start full stack
make cloud-up
bash scripts/healthcheck-cloud.sh
```

---

## §5 — Rollback to Pre-Cutover State — IRRELEVANT

> service-1 was never deployed. There is no pre-cutover service-1 state to restore.
> If the NiteOS stack fails and cannot be recovered, take the VPS provider snapshot rollback path.

### Original service-1 rollback instructions (kept for pattern reference only)

Use when: NiteOS is not healthy after cutover and you need to restore service-1 immediately.

```bash
# Stop NiteOS stack (or just Traefik to free port 80/443)
docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env \
  stop traefik

# Restart service-1
systemctl start service-1   # adjust to actual service-1 start command
# or:
# docker start <service-1-container>
# systemctl start nginx

# Confirm service-1 is responding
curl -s -o /dev/null -w "%{http_code}" http://localhost/
```

After service-1 is confirmed healthy:
- Diagnose NiteOS issue at leisure (logs: `make cloud-logs`)
- Fix, re-test on staging (port 8443), then attempt cutover again

---

## §6 — Emergency Contacts and Escalation

| Situation | Action |
|-----------|--------|
| Database restore fails | Restore from latest rclone/S3 remote backup; re-run migrations |
| JWT private key lost | Generate new key pair; all active sessions are invalidated; users must re-login |
| `acme.json` deleted | Delete and recreate with `touch + chmod 600`; Traefik re-issues cert (Let's Encrypt rate limit: 5 certs/domain/week) |
| `cloud.env` lost | Regenerate all secrets; all service credentials change; all tokens invalidated |
| VPS unrecoverable | Provision new VPS, restore from backup, update DNS A record |
