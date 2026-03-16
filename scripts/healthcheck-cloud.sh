#!/usr/bin/env bash
# NiteOS cloud health-check script.
# Hits /healthz on every service via the gateway and directly on internal ports.
# Run after `make cloud-up` to confirm all services are live.
#
# Usage:
#   bash scripts/healthcheck-cloud.sh                  # local (port-forwarded or dev)
#   DOMAIN=niteos.io bash scripts/healthcheck-cloud.sh # production via public endpoints
set -euo pipefail

DOMAIN="${DOMAIN:-}"
COMPOSE="docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env"
FAIL=0

green() { echo -e "\033[32m[ OK ]\033[0m $*"; }
red()   { echo -e "\033[31m[FAIL]\033[0m $*"; FAIL=1; }
info()  { echo "       $*"; }

check_url() {
  local label="$1" url="$2"
  local status
  status=$(curl -sk -o /dev/null -w "%{http_code}" --max-time 5 "$url" 2>/dev/null || echo "000")
  if [[ "$status" == "200" ]]; then
    green "$label → $url ($status)"
  else
    red "$label → $url (HTTP $status)"
  fi
}

echo ""
echo "=== NiteOS cloud health check ==="
echo ""

# ── Docker container status ────────────────────────────────────────────────────
echo "--- Container status ---"
$COMPOSE ps --format "table {{.Name}}\t{{.Status}}" 2>/dev/null || true
echo ""

# ── Internal healthz via docker exec (always works on VPS) ────────────────────
echo "--- Internal healthz (via docker exec) ---"
SERVICES=(
  "gateway:8000"
  "auth:8010"
  "profiles:8020"
  "ledger:8030"
  "wallet:8040"
  "ticketing:8050"
  "payments:8060"
  "devices:8070"
  "catalog:8080"
  "orders:8090"
  "sessions:8100"
  "sync:8110"
  "reporting:8120"
)
for entry in "${SERVICES[@]}"; do
  svc="${entry%%:*}"
  port="${entry##*:}"
  result=$($COMPOSE exec -T "$svc" wget -qO- "http://localhost:${port}/healthz" 2>/dev/null || echo "ERROR")
  if echo "$result" | grep -q '"status":"ok"'; then
    green "$svc (:$port /healthz)"
  else
    red "$svc (:$port /healthz) — $result"
  fi
done
echo ""

# ── Metrics endpoint spot check ────────────────────────────────────────────────
echo "--- Metrics endpoint spot check ---"
for svc_port in "gateway:8000" "auth:8010" "catalog:8080"; do
  svc="${svc_port%%:*}"
  port="${svc_port##*:}"
  result=$($COMPOSE exec -T "$svc" wget -qO- "http://localhost:${port}/metrics" 2>/dev/null | head -1 || echo "ERROR")
  if echo "$result" | grep -q "service_up"; then
    green "$svc /metrics"
  else
    red "$svc /metrics — $result"
  fi
done
echo ""

# ── Public endpoint checks (if DOMAIN is set) ─────────────────────────────────
if [[ -n "$DOMAIN" ]]; then
  echo "--- Public endpoints (https://$DOMAIN) ---"
  check_url "gateway /healthz"  "https://api.${DOMAIN}/healthz"
  check_url "admin-web /health" "https://admin.${DOMAIN}/api/health"
  check_url "grafana /health"   "https://grafana.${DOMAIN}/api/health"
  echo ""
fi

# ── Prometheus targets ─────────────────────────────────────────────────────────
echo "--- Prometheus target status ---"
PROM_TARGETS=$($COMPOSE exec -T prometheus wget -qO- "http://localhost:9090/api/v1/targets" 2>/dev/null || echo "ERROR")
if echo "$PROM_TARGETS" | grep -q '"status":"up"'; then
  UP_COUNT=$(echo "$PROM_TARGETS" | grep -o '"status":"up"' | wc -l | tr -d ' ')
  DOWN_COUNT=$(echo "$PROM_TARGETS" | grep -o '"status":"down"' | wc -l | tr -d ' ')
  if [[ "$DOWN_COUNT" -gt 0 ]]; then
    red "Prometheus: $UP_COUNT up, $DOWN_COUNT down"
  else
    green "Prometheus: $UP_COUNT targets up"
  fi
else
  red "Prometheus API unreachable"
fi
echo ""

# ── Summary ───────────────────────────────────────────────────────────────────
if [[ $FAIL -eq 0 ]]; then
  echo -e "\033[32m✓ All health checks passed\033[0m"
  echo ""
  exit 0
else
  echo -e "\033[31m✗ Some health checks FAILED — see output above\033[0m"
  echo "  Tip: docker compose ... logs <service> for details"
  echo ""
  exit 1
fi
