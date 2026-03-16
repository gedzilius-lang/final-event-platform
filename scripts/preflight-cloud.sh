#!/usr/bin/env bash
# NiteOS VPS A pre-flight check.
# Run before `make cloud-up` to catch missing config, secrets, and files.
# Exit 0 = ready; Exit 1 = one or more problems found.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${REPO_ROOT}/infra/cloud.env"
FAIL=0

red()   { echo -e "\033[31m[FAIL]\033[0m $*"; FAIL=1; }
ok()    { echo -e "\033[32m[ OK ]\033[0m $*"; }
warn()  { echo -e "\033[33m[WARN]\033[0m $*"; }
info()  { echo "       $*"; }

echo ""
echo "=== NiteOS VPS A pre-flight check ==="
echo ""

# ── 1. cloud.env exists ────────────────────────────────────────────────────────
if [[ ! -f "$ENV_FILE" ]]; then
  red "infra/cloud.env not found"
  info "Run: cp infra/cloud.env.example infra/cloud.env && nano infra/cloud.env"
  echo ""
  echo "Pre-flight FAILED — cannot continue without cloud.env"
  exit 1
fi
ok "infra/cloud.env exists"

# Load env
set -a; source "$ENV_FILE"; set +a

# ── 2. Required env vars ───────────────────────────────────────────────────────
REQUIRED_VARS=(
  DOMAIN ACME_EMAIL
  CF_DNS_API_TOKEN TRAEFIK_DASHBOARD_USERS
  POSTGRES_PASSWORD REDIS_PASSWORD
  STRIPE_API_KEY STRIPE_WEBHOOK_SECRET
  SESSION_SECRET GRAFANA_ADMIN_PASSWORD
  TRAEFIK_PORT_SUFFIX
)

for var in "${REQUIRED_VARS[@]}"; do
  val="${!var:-}"
  if [[ -z "$val" ]]; then
    # TRAEFIK_PORT_SUFFIX is intentionally empty in production (standard ports 80/443)
    if [[ "$var" == "TRAEFIK_PORT_SUFFIX" ]]; then
      ok "$var is empty (production mode: Traefik on :80/:443)"
    else
      red "env var $var is not set in cloud.env"
    fi
  elif [[ "$val" == *CHANGEME* ]] || [[ "$val" == *REPLACE_WITH* ]]; then
    red "env var $var still has placeholder value: $val"
  else
    ok "$var is set"
  fi
done

# ── 3. Password strength (≥ 20 chars after base64) ────────────────────────────
for var in POSTGRES_PASSWORD REDIS_PASSWORD SESSION_SECRET GRAFANA_ADMIN_PASSWORD; do
  val="${!var:-}"
  if [[ ${#val} -lt 20 ]]; then
    warn "$var is short (${#val} chars) — recommend openssl rand -base64 32"
  fi
done

# ── 4. JWT keys ────────────────────────────────────────────────────────────────
if [[ ! -f "$REPO_ROOT/infra/secrets/jwt_private_key.pem" ]]; then
  red "infra/secrets/jwt_private_key.pem not found"
  info "Run: openssl genrsa -out infra/secrets/jwt_private_key.pem 2048"
else
  ok "jwt_private_key.pem exists"
  PERMS=$(stat -c "%a" "$REPO_ROOT/infra/secrets/jwt_private_key.pem" 2>/dev/null || stat -f "%A" "$REPO_ROOT/infra/secrets/jwt_private_key.pem" 2>/dev/null || echo "unknown")
  if [[ "$PERMS" != "600" ]] && [[ "$PERMS" != "0600" ]]; then
    warn "jwt_private_key.pem permissions are $PERMS (recommend 600)"
    info "Run: chmod 600 infra/secrets/jwt_private_key.pem"
  fi
fi

if [[ ! -f "$REPO_ROOT/infra/secrets/jwt_public_key.pem" ]]; then
  red "infra/secrets/jwt_public_key.pem not found"
  info "Run: openssl rsa -in infra/secrets/jwt_private_key.pem -pubout -out infra/secrets/jwt_public_key.pem"
else
  ok "jwt_public_key.pem exists"
fi

# ── 5. Traefik ACME storage ────────────────────────────────────────────────────
ACME_FILE="$REPO_ROOT/infra/traefik/acme.json"
if [[ ! -f "$ACME_FILE" ]]; then
  red "infra/traefik/acme.json not found"
  info "Run: touch infra/traefik/acme.json && chmod 600 infra/traefik/acme.json"
else
  ACME_PERMS=$(stat -c "%a" "$ACME_FILE" 2>/dev/null || stat -f "%A" "$ACME_FILE" 2>/dev/null || echo "unknown")
  if [[ "$ACME_PERMS" != "600" ]] && [[ "$ACME_PERMS" != "0600" ]]; then
    red "infra/traefik/acme.json permissions are $ACME_PERMS — Traefik requires exactly 600"
    info "Run: chmod 600 infra/traefik/acme.json"
  else
    ok "infra/traefik/acme.json exists with mode 600"
  fi
fi

# ── 6. Docker available ────────────────────────────────────────────────────────
if ! command -v docker &>/dev/null; then
  red "docker not found on PATH"
  info "Install: curl -fsSL https://get.docker.com | sh"
else
  ok "docker $(docker --version | awk '{print $3}' | tr -d ',')"
fi

if ! docker compose version &>/dev/null; then
  red "docker compose (v2) not available"
  info "Ensure Docker Engine >= 23 or install compose plugin"
else
  ok "docker compose $(docker compose version --short)"
fi

# ── 7. Required config files ────────────────────────────────────────────────────
REQUIRED_FILES=(
  "infra/traefik/traefik.yml"
  "infra/traefik/dynamic/routes.yml"
  "infra/prometheus.yml"
  "infra/docker-compose.cloud.yml"
  "infra/grafana/provisioning/datasources/prometheus.yml"
  "infra/grafana/provisioning/dashboards/dashboards.yml"
)
for f in "${REQUIRED_FILES[@]}"; do
  if [[ ! -f "$REPO_ROOT/$f" ]]; then
    red "Missing required file: $f"
  else
    ok "$f"
  fi
done

# ── 8. Script executability ────────────────────────────────────────────────────
for s in scripts/migrate.sh scripts/backup.sh scripts/restore.sh; do
  if [[ ! -x "$REPO_ROOT/$s" ]]; then
    warn "$s is not executable — run: chmod +x $s"
  fi
done

# ── 9. Stripe keys staging warning ────────────────────────────────────────────
if [[ "${STRIPE_API_KEY:-}" == sk_test_* ]]; then
  warn "STRIPE_API_KEY is a test key — use sk_live_ for production"
fi

# ── Summary ────────────────────────────────────────────────────────────────────
echo ""
if [[ $FAIL -eq 0 ]]; then
  echo -e "\033[32m✓ Pre-flight passed — safe to run: make cloud-up\033[0m"
  echo ""
  exit 0
else
  echo -e "\033[31m✗ Pre-flight FAILED — fix the issues above before deploying\033[0m"
  echo ""
  exit 1
fi
