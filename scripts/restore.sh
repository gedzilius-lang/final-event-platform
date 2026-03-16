#!/usr/bin/env bash
# NiteOS Postgres restore script — VPS A
# Restores from a local .sql.gz backup file.
#
# Usage:
#   DATABASE_URL=<url> ./scripts/restore.sh /opt/niteos/backups/niteos_20260315_020001.sql.gz
#
# WARNING: This DROPS and recreates the niteos database. Only run on a stopped stack.
#          Stop services first: make cloud-down (or docker compose -f infra/docker-compose.cloud.yml down)
set -euo pipefail

BACKUP_FILE="${1:-}"
if [[ -z "$BACKUP_FILE" ]]; then
  echo "Usage: $0 <backup-file.sql.gz>" >&2
  exit 1
fi

if [[ ! -f "$BACKUP_FILE" ]]; then
  echo "ERROR: backup file not found: $BACKUP_FILE" >&2
  exit 1
fi

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "ERROR: DATABASE_URL is not set" >&2
  exit 1
fi

# Derive admin connection URL (connect to postgres database to drop/create niteos)
ADMIN_URL="${DATABASE_URL/\/niteos?/\/postgres?}"

echo "[restore] WARNING: this will DROP the niteos database and restore from $BACKUP_FILE"
read -p "Type 'yes' to continue: " CONFIRM
if [[ "$CONFIRM" != "yes" ]]; then
  echo "[restore] aborted"
  exit 1
fi

echo "[restore] dropping and recreating niteos database..."
psql "$ADMIN_URL" -c "DROP DATABASE IF EXISTS niteos;"
psql "$ADMIN_URL" -c "CREATE DATABASE niteos OWNER niteos;"

echo "[restore] restoring from $BACKUP_FILE..."
gunzip -c "$BACKUP_FILE" | psql "$DATABASE_URL"

echo "[restore] complete — verify with: docker compose -f infra/docker-compose.cloud.yml up -d"
