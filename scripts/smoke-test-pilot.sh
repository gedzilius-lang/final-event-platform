#!/usr/bin/env bash
# NiteOS pilot smoke test.
# Exercises core flows end-to-end against a running cloud stack.
# Safe to run against staging or production — creates test data with unique IDs,
# does NOT run database cleanup (test venue/user remain; delete via admin console).
#
# Requirements: curl, jq
#
# Usage:
#   API_BASE=https://api.niteos.io \
#   NITECORE_EMAIL=nitecore@niteos.io \
#   NITECORE_PASSWORD=<password> \
#   bash scripts/smoke-test-pilot.sh
set -euo pipefail

API_BASE="${API_BASE:-https://api.niteos.io}"
NITECORE_EMAIL="${NITECORE_EMAIL:-nitecore@niteos.io}"
NITECORE_PASSWORD="${NITECORE_PASSWORD:-}"
FAIL=0

if ! command -v jq &>/dev/null; then
  echo "ERROR: jq is required — apt-get install jq" >&2
  exit 1
fi

if [[ -z "$NITECORE_PASSWORD" ]]; then
  echo "ERROR: NITECORE_PASSWORD is required" >&2
  echo "Usage: NITECORE_EMAIL=... NITECORE_PASSWORD=... bash $0" >&2
  exit 1
fi

TS=$(date +%s)
green() { echo -e "\033[32m[ OK ]\033[0m $*"; }
red()   { echo -e "\033[31m[FAIL]\033[0m $*"; FAIL=1; }
step()  { echo -e "\n\033[1m--- $* ---\033[0m"; }

api() {
  local method="$1" path="$2" token="${3:-}" body="${4:-}"
  local args=(-s -X "$method" "${API_BASE}${path}" -H "Content-Type: application/json")
  [[ -n "$token" ]] && args+=(-H "Authorization: Bearer $token")
  [[ -n "$body" ]] && args+=(-d "$body")
  curl "${args[@]}"
}

# ── 1. Gateway health ──────────────────────────────────────────────────────────
step "1. Gateway health"
HEALTH=$(api GET /healthz)
if echo "$HEALTH" | jq -e '.status == "ok"' >/dev/null 2>&1; then
  green "gateway /healthz"
else
  red "gateway /healthz — $HEALTH"
fi

# ── 2. Nitecore login ──────────────────────────────────────────────────────────
step "2. Nitecore login"
LOGIN=$(api POST /auth/login "" "{\"email\":\"$NITECORE_EMAIL\",\"password\":\"$NITECORE_PASSWORD\"}")
NC_TOKEN=$(echo "$LOGIN" | jq -r '.access_token // empty')
if [[ -z "$NC_TOKEN" ]]; then
  red "nitecore login failed — $LOGIN"
  echo "Cannot continue without nitecore token"
  exit 1
fi
green "nitecore login (got access_token)"

# ── 3. Create test venue ───────────────────────────────────────────────────────
step "3. Create test venue"
VENUE_SLUG="smoke-test-${TS}"
VENUE_RESP=$(api POST /catalog/venues "$NC_TOKEN" \
  "{\"name\":\"Smoke Test Venue $TS\",\"slug\":\"$VENUE_SLUG\",\"city\":\"TestCity\",\"capacity\":50,\"staff_pin\":\"9999\",\"timezone\":\"Europe/Zurich\"}")
VENUE_ID=$(echo "$VENUE_RESP" | jq -r '.venue_id // empty')
if [[ -z "$VENUE_ID" ]]; then
  red "create venue failed — $VENUE_RESP"
else
  green "created venue (id=$VENUE_ID)"
fi

# ── 4. Create catalog item ─────────────────────────────────────────────────────
step "4. Create catalog item"
ITEM_RESP=$(api POST "/catalog/venues/${VENUE_ID}/items" "$NC_TOKEN" \
  "{\"name\":\"Test Beer\",\"category\":\"drinks\",\"price_nc\":800,\"icon\":\"🍺\",\"display_order\":1}")
ITEM_ID=$(echo "$ITEM_RESP" | jq -r '.item_id // empty')
if [[ -z "$ITEM_ID" ]]; then
  red "create item failed — $ITEM_RESP"
else
  green "created catalog item (id=$ITEM_ID)"
fi

# ── 5. List venues ─────────────────────────────────────────────────────────────
step "5. List venues"
VENUES=$(api GET /catalog/venues "$NC_TOKEN")
COUNT=$(echo "$VENUES" | jq '.venues | length' 2>/dev/null || echo 0)
if [[ "$COUNT" -gt 0 ]]; then
  green "list venues returned $COUNT venue(s)"
else
  red "list venues empty or failed — $VENUES"
fi

# ── 6. Register test guest user ────────────────────────────────────────────────
step "6. Register test guest user"
GUEST_EMAIL="smoke-guest-${TS}@test.niteos.io"
GUEST_RESP=$(api POST /auth/register "" \
  "{\"email\":\"$GUEST_EMAIL\",\"password\":\"TestPass123!\",\"display_name\":\"Smoke Guest $TS\"}")
GUEST_ID=$(echo "$GUEST_RESP" | jq -r '.user_id // empty')
GUEST_TOKEN=$(echo "$GUEST_RESP" | jq -r '.access_token // empty')
if [[ -z "$GUEST_ID" ]]; then
  red "register guest failed — $GUEST_RESP"
else
  green "registered guest (id=$GUEST_ID)"
fi

# ── 7. Guest login ─────────────────────────────────────────────────────────────
step "7. Guest login"
GUEST_LOGIN=$(api POST /auth/login "" \
  "{\"email\":\"$GUEST_EMAIL\",\"password\":\"TestPass123!\"}")
GUEST_TOKEN=$(echo "$GUEST_LOGIN" | jq -r '.access_token // empty')
if [[ -z "$GUEST_TOKEN" ]]; then
  red "guest login failed — $GUEST_LOGIN"
else
  green "guest login ok"
fi

# ── 8. Guest wallet balance ────────────────────────────────────────────────────
step "8. Guest wallet balance (expect 0 NC)"
BALANCE=$(api GET "/wallet/${GUEST_ID}" "$GUEST_TOKEN")
BAL_NC=$(echo "$BALANCE" | jq -r '.balance_nc // .balance // empty')
if [[ -n "$BAL_NC" ]]; then
  green "wallet balance: ${BAL_NC} NC"
else
  red "wallet balance failed — $BALANCE"
fi

# ── 9. List catalog items for venue ───────────────────────────────────────────
step "9. List catalog items"
ITEMS=$(api GET "/catalog/venues/${VENUE_ID}/items" "$NC_TOKEN")
ITEM_COUNT=$(echo "$ITEMS" | jq '.items | length' 2>/dev/null || echo 0)
if [[ "$ITEM_COUNT" -gt 0 ]]; then
  green "catalog items: $ITEM_COUNT item(s)"
else
  red "catalog items empty — $ITEMS"
fi

# ── 10. Reporting endpoint (no sessions yet — expect empty) ───────────────────
step "10. Reporting revenue endpoint"
TODAY=$(date -u +%Y-%m-%d)
REVENUE=$(api GET "/reporting/venues/${VENUE_ID}/revenue?from=2026-01-01&to=${TODAY}" "$NC_TOKEN")
if echo "$REVENUE" | jq -e '.venue_id' >/dev/null 2>&1; then
  green "reporting revenue endpoint ok"
else
  red "reporting revenue failed — $REVENUE"
fi

# ── Summary ────────────────────────────────────────────────────────────────────
echo ""
echo "=== Smoke test results ==="
echo "  Test venue ID : $VENUE_ID"
echo "  Test venue slug: $VENUE_SLUG"
echo "  Test guest ID  : $GUEST_ID"
echo "  Test guest email: $GUEST_EMAIL"
echo ""
echo "  NOTE: test data was NOT cleaned up — delete via admin console if needed."
echo ""
if [[ $FAIL -eq 0 ]]; then
  echo -e "\033[32m✓ All smoke tests passed — stack is pilot-ready\033[0m"
  exit 0
else
  echo -e "\033[31m✗ Some smoke tests FAILED — review output above\033[0m"
  exit 1
fi
