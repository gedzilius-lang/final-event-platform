#!/usr/bin/env bash
set -euo pipefail
echo "Starting NiteOS dev stack (Postgres + Redis)..."
docker compose -f infra/docker-compose.dev.yml up -d
echo "Waiting for Postgres..."
until docker compose -f infra/docker-compose.dev.yml exec -T postgres pg_isready -U niteos >/dev/null 2>&1; do
  sleep 1
done
echo "Running migrations..."
bash scripts/migrate.sh
echo "Dev stack ready."
echo "  Postgres: localhost:5432 (db=niteos, user=niteos, password=devpassword)"
echo "  Redis:    localhost:6379 (password=devpassword)"
