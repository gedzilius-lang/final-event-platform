#!/usr/bin/env bash
# bootstrap-nitecore.sh — Create the first nitecore user in a fresh NiteOS database.
#
# This is a one-time operation for first deployment. It inserts the nitecore superuser
# directly into Postgres using a bcrypt-hashed password. Run AFTER migrations, BEFORE
# starting the full stack (or with only the postgres service running).
#
# Requirements: docker (for compose exec), python3 with bcrypt OR htpasswd available
#
# Usage:
#   NITECORE_EMAIL=nitecore@peoplewelike.club \
#   NITECORE_PASSWORD=<strong_password> \
#   NITECORE_DISPLAY_NAME="NiteOS Admin" \
#   bash scripts/bootstrap-nitecore.sh
#
# Or with custom DATABASE_URL (host psql):
#   DATABASE_URL=postgres://niteos:<pw>@localhost:5432/niteos?sslmode=disable \
#   NITECORE_EMAIL=... NITECORE_PASSWORD=... bash scripts/bootstrap-nitecore.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

NITECORE_EMAIL="${NITECORE_EMAIL:-}"
NITECORE_PASSWORD="${NITECORE_PASSWORD:-}"
NITECORE_DISPLAY_NAME="${NITECORE_DISPLAY_NAME:-NiteOS Admin}"
DATABASE_URL="${DATABASE_URL:-}"

red()   { echo -e "\033[31m[FAIL]\033[0m $*" >&2; }
green() { echo -e "\033[32m[ OK ]\033[0m $*"; }
info()  { echo "       $*"; }

if [[ -z "$NITECORE_EMAIL" ]]; then
  red "NITECORE_EMAIL is required"
  echo "Usage: NITECORE_EMAIL=... NITECORE_PASSWORD=... bash $0" >&2
  exit 1
fi
if [[ -z "$NITECORE_PASSWORD" ]]; then
  red "NITECORE_PASSWORD is required"
  echo "Usage: NITECORE_EMAIL=... NITECORE_PASSWORD=... bash $0" >&2
  exit 1
fi
if [[ ${#NITECORE_PASSWORD} -lt 16 ]]; then
  red "NITECORE_PASSWORD must be at least 16 characters"
  exit 1
fi

echo ""
echo "=== NiteOS nitecore bootstrap ==="
echo "  Email: $NITECORE_EMAIL"
echo "  Name:  $NITECORE_DISPLAY_NAME"
echo ""

# ── Generate bcrypt hash ───────────────────────────────────────────────────────
# Try python3 bcrypt first, fall back to htpasswd
if python3 -c "import bcrypt" 2>/dev/null; then
  HASH=$(python3 -c "
import bcrypt, sys
pw = sys.argv[1].encode()
print(bcrypt.hashpw(pw, bcrypt.gensalt(12)).decode())
" "$NITECORE_PASSWORD")
  green "bcrypt hash generated (python3)"
elif command -v htpasswd &>/dev/null; then
  # htpasswd -bnB '' password | cut -d: -f2 gives the bcrypt hash
  HASH=$(htpasswd -bnB '' "$NITECORE_PASSWORD" | cut -d: -f2)
  green "bcrypt hash generated (htpasswd)"
else
  red "Neither python3 bcrypt nor htpasswd is available."
  info "Install one: pip3 install bcrypt  OR  apt-get install apache2-utils"
  exit 1
fi

# ── Execute SQL ───────────────────────────────────────────────────────────────
SQL="INSERT INTO profiles.users (email, password_hash, display_name, role)
     VALUES ('${NITECORE_EMAIL}', '${HASH}', '${NITECORE_DISPLAY_NAME}', 'nitecore')
     ON CONFLICT (email) DO UPDATE
       SET password_hash = EXCLUDED.password_hash,
           display_name  = EXCLUDED.display_name,
           role          = 'nitecore',
           updated_at    = now()
     RETURNING user_id, email, role;"

if [[ -n "$DATABASE_URL" ]]; then
  # Use host psql with DATABASE_URL
  if ! command -v psql &>/dev/null; then
    red "psql not found — install postgresql-client or use docker compose exec"
    exit 1
  fi
  RESULT=$(psql "$DATABASE_URL" -tAc "$SQL")
  green "Inserted via host psql"
else
  # Default: use docker compose exec (postgres container must be running)
  ENV_FILE="${REPO_ROOT}/infra/cloud.env"
  if [[ ! -f "$ENV_FILE" ]]; then
    red "infra/cloud.env not found and DATABASE_URL not set"
    info "Either set DATABASE_URL or run from repo root with infra/cloud.env present"
    exit 1
  fi
  RESULT=$(docker compose \
    -f "${REPO_ROOT}/infra/docker-compose.cloud.yml" \
    --env-file "$ENV_FILE" \
    exec -T postgres \
    psql -U niteos -d niteos -tAc "$SQL")
  green "Inserted via docker compose exec"
fi

echo ""
echo "=== Result ==="
echo "$RESULT"
echo ""
green "Nitecore user created/updated. Login at https://admin.peoplewelike.club"
echo ""
echo "  Email:    $NITECORE_EMAIL"
echo "  Password: (as provided)"
echo ""
echo "  Next step: bash scripts/pilot-bootstrap.sh"
echo ""
