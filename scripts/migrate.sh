#!/usr/bin/env bash
# Run all Postgres migrations for NiteOS.
# Uses psql directly — no golang-migrate binary required.
# Files are applied in numeric order within each schema directory.
# Requires: psql on PATH, dev stack running (make dev-up).
set -euo pipefail

DB="${DATABASE_URL:-postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable}"

echo "==> 000_init_schemas.sql"
psql "$DB" -f migrations/000_init_schemas.sql

for schema in profiles ledger payments ticketing orders catalog devices sessions sync; do
  dir="migrations/$schema"
  if [ ! -d "$dir" ]; then continue; fi
  files=$(ls "$dir"/*.sql 2>/dev/null | sort) || continue
  for f in $files; do
    echo "==> $f"
    psql "$DB" -f "$f"
  done
done

echo ""
echo "Migrations complete."
