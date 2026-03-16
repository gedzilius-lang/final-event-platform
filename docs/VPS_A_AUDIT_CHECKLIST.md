# VPS A Audit Checklist

Run these commands on VPS A **before** making any deployment decision.
Copy output into a text file — you will need it to fill in the decision framework.

---

## 1. OS and Hardware

```bash
# OS version and kernel
lsb_release -a
uname -r

# CPU / RAM
nproc
free -h

# Disk usage
df -h

# Block devices
lsblk
```

---

## 2. Installed Packages (relevant ones)

```bash
# Docker
docker --version 2>/dev/null || echo "docker: NOT INSTALLED"
docker compose version 2>/dev/null || echo "docker compose: NOT INSTALLED"

# nginx
nginx -v 2>/dev/null || echo "nginx: NOT INSTALLED"

# PostgreSQL (host-level, if any)
psql --version 2>/dev/null || echo "psql: NOT INSTALLED"
pg_lsclusters 2>/dev/null || echo "pg_lsclusters: not available"

# Redis (host-level, if any)
redis-cli --version 2>/dev/null || echo "redis-cli: NOT INSTALLED"

# Other relevant binaries
which go node npm python3 certbot traefik 2>/dev/null || true

# Snap packages
snap list 2>/dev/null || echo "snap: not available"

# List all manually installed packages (large — pipe to less)
apt list --installed 2>/dev/null | grep -v automatic | less
```

---

## 3. Running Services (systemd)

```bash
# All active services
systemctl list-units --type=service --state=running --no-pager

# All enabled services (starts at boot)
systemctl list-unit-files --type=service --state=enabled --no-pager

# Timers
systemctl list-timers --all --no-pager

# Check for nginx, postgres, redis, traefik
for svc in nginx postgresql redis traefik docker; do
  systemctl is-active $svc 2>/dev/null && echo "$svc: ACTIVE" || echo "$svc: inactive/absent"
done
```

---

## 4. Docker State

```bash
# Is Docker running?
systemctl is-active docker

# Docker info (version, storage driver, network)
docker info 2>/dev/null || echo "Docker not running"

# Running containers
docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}"

# ALL containers (including stopped)
docker ps -a --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"

# Docker networks
docker network ls

# Docker volumes
docker volume ls

# Docker images on disk
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"

# Disk usage by Docker
docker system df
```

---

## 5. Open Ports and Network

```bash
# Listening ports (all protocols)
ss -tlnp
# or:
netstat -tlnp 2>/dev/null || echo "netstat not available"

# Firewall rules
ufw status verbose 2>/dev/null || iptables -L -n --line-numbers 2>/dev/null

# Public IP
curl -s https://ifconfig.me; echo
```

---

## 6. Reverse Proxy Config

```bash
# nginx — check if installed and what it's serving
nginx -t 2>/dev/null
ls /etc/nginx/sites-enabled/ 2>/dev/null
cat /etc/nginx/sites-enabled/default 2>/dev/null || echo "no default nginx site"
ls /etc/nginx/conf.d/ 2>/dev/null

# Traefik — host-level install
which traefik && traefik version 2>/dev/null || echo "traefik: not on PATH"
ls /etc/traefik/ 2>/dev/null || echo "/etc/traefik: absent"

# Caddy
which caddy && caddy version 2>/dev/null || echo "caddy: absent"

# Certificates (Let's Encrypt)
ls /etc/letsencrypt/live/ 2>/dev/null || echo "certbot: no live certs"
ls /root/.acme.sh/ 2>/dev/null || echo "acme.sh: absent"
```

---

## 7. Existing Repo and Application Files

```bash
# Expected NiteOS deploy path
ls -la /opt/niteos/ 2>/dev/null || echo "/opt/niteos: absent"

# Any other app directories
ls /opt/ 2>/dev/null
ls /var/www/ 2>/dev/null
ls /srv/ 2>/dev/null
ls /home/ 2>/dev/null

# Git repos
find /opt /var/www /srv /root -maxdepth 3 -name ".git" -type d 2>/dev/null

# Current NiteOS git state (if repo exists)
if [ -d /opt/niteos/.git ]; then
  cd /opt/niteos && git log --oneline -5 && git status --short
fi
```

---

## 8. Environment Files and Secrets

```bash
# NiteOS env files (NEVER print contents — just confirm presence)
for f in \
  /opt/niteos/infra/cloud.env \
  /opt/niteos/infra/secrets/jwt_private_key.pem \
  /opt/niteos/infra/secrets/jwt_public_key.pem \
  /opt/niteos/infra/traefik/acme.json \
  /opt/niteos/backup.env; do
  if [ -f "$f" ]; then
    echo "EXISTS  $(stat -c '%a %n' $f)"
  else
    echo "ABSENT  $f"
  fi
done
```

---

## 9. Databases

```bash
# Host-level Postgres (if any)
if command -v psql &>/dev/null; then
  sudo -u postgres psql -c "\l" 2>/dev/null || echo "postgres: no local cluster"
fi

# Dockerised Postgres (if Docker is up)
COMPOSE="docker compose -f /opt/niteos/infra/docker-compose.cloud.yml --env-file /opt/niteos/infra/cloud.env"
$COMPOSE exec postgres psql -U niteos -d niteos -c "\dn" 2>/dev/null || echo "docker postgres: not reachable"

# Postgres data volume
docker volume inspect niteos_pgdata 2>/dev/null || echo "pgdata volume: absent"
```

---

## 10. Docker Volumes and Bind Mounts

```bash
# All Docker volumes and their mount points
docker volume ls -q | xargs -I{} docker volume inspect {} --format '{{.Name}} → {{.Mountpoint}} ({{.Driver}})' 2>/dev/null

# Disk usage per volume (may require root)
du -sh /var/lib/docker/volumes/*/  2>/dev/null | sort -rh | head -20
```

---

## 11. Backup Targets

```bash
# Local backups
ls -lh /opt/niteos/backups/ 2>/dev/null || echo "/opt/niteos/backups: absent"

# Systemd backup timer
systemctl status niteos-backup.timer 2>/dev/null || echo "niteos-backup.timer: absent"
systemctl status niteos-backup.service 2>/dev/null || echo "niteos-backup.service: absent"

# Backup env
[ -f /opt/niteos/backup.env ] && echo "backup.env: present" || echo "backup.env: absent"

# Remote target (rclone config, if set)
rclone listremotes 2>/dev/null || echo "rclone: not installed or not configured"
```

---

## 12. Disk Usage Summary

```bash
# Top-level disk usage
df -h /

# Largest directories
du -sh /opt/* /var/lib/docker /var/log /home/* 2>/dev/null | sort -rh | head -20

# Docker-specific usage
docker system df -v 2>/dev/null
```

---

## 13. service-1 (existing service on VPS A)

```bash
# Identify what service-1 is
ls /opt/service-1/ 2>/dev/null || echo "no /opt/service-1"

# Any nginx vhosts pointing to service-1
grep -r "service-1\|service1" /etc/nginx/ 2>/dev/null || echo "no nginx config referencing service-1"

# Process list — find anything running on 80/443
ss -tlnp | grep -E ':80|:443'
lsof -i :80 -i :443 2>/dev/null | grep LISTEN
```

---

## 14. Record Summary

After running the above, fill in the following table before proceeding to the rebuild-vs-overwrite decision:

| Item | Value |
|------|-------|
| OS version | |
| Kernel | |
| Docker installed | yes / no |
| Docker version | |
| nginx running | yes / no |
| nginx ports | |
| Traefik (host) | yes / no |
| `/opt/niteos` exists | yes / no |
| NiteOS repo commit | |
| `cloud.env` present | yes / no |
| JWT keys present | yes / no |
| `acme.json` present | yes / no |
| Docker pgdata volume | yes / no |
| Postgres schemas present | yes / no |
| Local backups present | yes / no |
| Disk free on `/` | |
| service-1 running | yes / no |
| service-1 port | |

Item	Value
OS version	Ubuntu 22.04.5 LTS
Kernel	5.15.0-164-generic
Docker installed	yes
Docker version	29.2.1
nginx running	yes
nginx ports	80, 443, 8088, 8089
Traefik (host)	no
/opt/niteos exists	no
NiteOS repo commit	N/A
cloud.env present	no
JWT keys present	no
acme.json present	no
Docker pgdata volume	no
Postgres schemas present	no
Local backups present	no
Disk free on /	35G
service-1 running	no
service-1 port	N/A
