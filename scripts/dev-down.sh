#!/usr/bin/env bash
set -euo pipefail
docker compose -f infra/docker-compose.dev.yml down
echo "Dev stack stopped."
