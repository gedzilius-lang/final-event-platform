#!/usr/bin/env bash
# NiteOS pilot smoke test.
# Exercises core API flows end-to-end against a running cloud stack.
# Safe to run against staging or production — creates test data with unique IDs.
# Test data is NOT cleaned up (delete via admin console if needed).
#
# Tested flows:
#   1.  Gateway health
#   2.  Nitecore login
#   3.  Create test venue + catalog item
#   4.  List venues (catalog visible)
#   5.  Register test guest
#   6.  Guest login + wallet balance (0 NC on fresh account)
#   7.  Register door_staff + assign role
#   8.  Register bartender + assign role
#   9.  Door staff login + guest check-in
#   10. Verify active session exists (guest + manager view)
#   11. Bartender login + create order
#   12. Attempt finalize (expect 402 — no balance, flow is correct)
#   13. Reporting revenue endpoint
#
# Requirements: curl, jq
#
# Usage:
#   API_BASE=https://api.peoplewelike.club \
#   NITECORE_EMAIL=nitecore@peoplewelike.club \
#   NITECORE_PASSWORD=<password> \
#   bash scripts/smoke-test-pilot.sh
set -euo pipefail

API_BASE="${API_BASE:-https://api.peoplewelike.club}"
NITECORE_EMAIL="${NITECORE_EMAIL:-nitecore@peoplewelike.club}"
NITECORE_PASSWORD="${NITECORE_PASSWORD:-}"
FAIL=0

if ! command -v jq &>/dev/null; then
  echo "ERROR: jq is required — apt-get install jq" >&2; exit 1
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
info()  { echo "       $*"; }

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
NC_USER_ID=$(echo "$LOGIN" | jq -r '.user_id // empty')
if [[ -z "$NC_TOKEN" ]]; then
  red "nitecore login failed — $LOGIN"
  echo "Cannot continue without nitecore token." >&2
  exit 1
fi
green "nitecore login OK (user_id=$NC_USER_ID)"

# ── 3. Create test venue + catalog item ────────────────────────────────────────
step "3. Create test venue"
VENUE_SLUG="smoke-${TS}"
VENUE_RESP=$(api POST /catalog/venues "$NC_TOKEN" \
  "{\"name\":\"Smoke Venue $TS\",\"slug\":\"$VENUE_SLUG\",\"city\":\"TestCity\",\"capacity\":50,\"staff_pin\":\"9999\",\"timezone\":\"Europe/Zurich\"}")
VENUE_ID=$(echo "$VENUE_RESP" | jq -r '.venue_id // empty')
if [[ -z "$VENUE_ID" ]]; then
  red "create venue failed — $VENUE_RESP"
  echo "Cannot continue without venue_id." >&2; exit 1
fi
green "venue created (id=$VENUE_ID)"

ITEM_RESP=$(api POST "/catalog/venues/${VENUE_ID}/items" "$NC_TOKEN" \
  '{"name":"Test Beer","category":"drinks","price_nc":8,"icon":"🍺","display_order":1}')
ITEM_ID=$(echo "$ITEM_RESP" | jq -r '.item_id // empty')
if [[ -z "$ITEM_ID" ]]; then
  red "create catalog item failed — $ITEM_RESP"
else
  green "catalog item created: 8 NC (id=$ITEM_ID)"
fi

# ── 4. List venues ─────────────────────────────────────────────────────────────
step "4. List venues"
VENUES=$(api GET /catalog/venues "$NC_TOKEN")
COUNT=$(echo "$VENUES" | jq '.venues | length' 2>/dev/null || echo 0)
if [[ "$COUNT" -gt 0 ]]; then
  green "list venues returned $COUNT venue(s)"
else
  red "list venues empty or failed — $VENUES"
fi

# ── 5. Register test guest ─────────────────────────────────────────────────────
step "5. Register test guest"
GUEST_EMAIL="smoke-guest-${TS}@test.internal"
GUEST_RESP=$(api POST /auth/register "" \
  "{\"email\":\"$GUEST_EMAIL\",\"password\":\"TestPass123!\",\"display_name\":\"Smoke Guest $TS\"}")
GUEST_ID=$(echo "$GUEST_RESP" | jq -r '.user_id // empty')
if [[ -z "$GUEST_ID" ]]; then
  red "register guest failed — $GUEST_RESP"
  echo "Cannot continue without guest_id." >&2; exit 1
fi
green "guest registered (id=$GUEST_ID)"

# ── 6. Guest login + wallet balance ────────────────────────────────────────────
step "6. Guest login + wallet balance"
GUEST_LOGIN=$(api POST /auth/login "" \
  "{\"email\":\"$GUEST_EMAIL\",\"password\":\"TestPass123!\"}")
GUEST_TOKEN=$(echo "$GUEST_LOGIN" | jq -r '.access_token // empty')
if [[ -z "$GUEST_TOKEN" ]]; then
  red "guest login failed — $GUEST_LOGIN"
else
  green "guest login OK"
fi

BALANCE=$(api GET "/wallet/${GUEST_ID}" "$GUEST_TOKEN")
BAL_NC=$(echo "$BALANCE" | jq -r '.balance_nc // empty')
if [[ -n "$BAL_NC" ]]; then
  green "wallet balance: ${BAL_NC} NC (expect 0 for new guest)"
else
  red "wallet balance failed — $BALANCE"
fi

# ── 7. Register door_staff + assign role ───────────────────────────────────────
step "7. Register door_staff account"
DOOR_EMAIL="smoke-door-${TS}@test.internal"
DOOR_RESP=$(api POST /auth/register "" \
  "{\"email\":\"$DOOR_EMAIL\",\"password\":\"DoorPass123!\",\"display_name\":\"Smoke Door $TS\"}")
DOOR_ID=$(echo "$DOOR_RESP" | jq -r '.user_id // empty')
if [[ -z "$DOOR_ID" ]]; then
  red "register door_staff failed — $DOOR_RESP"
else
  green "door_staff registered (id=$DOOR_ID)"
fi

PATCH=$(api PATCH "/profiles/users/${DOOR_ID}/venue" "$NC_TOKEN" \
  "{\"venue_id\":\"$VENUE_ID\",\"role\":\"door_staff\"}")
DOOR_ROLE=$(echo "$PATCH" | jq -r '.role // empty')
if [[ "$DOOR_ROLE" == "door_staff" ]]; then
  green "door_staff role assigned"
else
  red "role assignment failed — $PATCH"
fi

# ── 8. Register bartender + assign role ───────────────────────────────────────
step "8. Register bartender account"
BAR_EMAIL="smoke-bar-${TS}@test.internal"
BAR_RESP=$(api POST /auth/register "" \
  "{\"email\":\"$BAR_EMAIL\",\"password\":\"BarPass123!\",\"display_name\":\"Smoke Bar $TS\"}")
BAR_ID=$(echo "$BAR_RESP" | jq -r '.user_id // empty')
if [[ -z "$BAR_ID" ]]; then
  red "register bartender failed — $BAR_RESP"
else
  green "bartender registered (id=$BAR_ID)"
fi

PATCH=$(api PATCH "/profiles/users/${BAR_ID}/venue" "$NC_TOKEN" \
  "{\"venue_id\":\"$VENUE_ID\",\"role\":\"bartender\"}")
BAR_ROLE=$(echo "$PATCH" | jq -r '.role // empty')
if [[ "$BAR_ROLE" == "bartender" ]]; then
  green "bartender role assigned"
else
  red "bartender role assignment failed — $PATCH"
fi

# ── 9. Door staff check-in guest ──────────────────────────────────────────────
step "9. Door staff check-in guest"
DOOR_LOGIN=$(api POST /auth/login "" \
  "{\"email\":\"$DOOR_EMAIL\",\"password\":\"DoorPass123!\"}")
DOOR_TOKEN=$(echo "$DOOR_LOGIN" | jq -r '.access_token // empty')
if [[ -z "$DOOR_TOKEN" ]]; then
  red "door_staff login failed — $DOOR_LOGIN"
else
  green "door_staff login OK (role=$(echo "$DOOR_LOGIN" | jq -r '.role // "?"'))"
fi

CHECKIN=$(api POST /sessions/checkin "$DOOR_TOKEN" \
  "{\"user_id\":\"$GUEST_ID\",\"venue_id\":\"$VENUE_ID\",\"nfc_uid\":\"\",\"device_id\":\"\"}")
SESSION_ID=$(echo "$CHECKIN" | jq -r '.session_id // empty')
SESSION_STATUS=$(echo "$CHECKIN" | jq -r '.status // empty')
if [[ -n "$SESSION_ID" && "$SESSION_STATUS" == "open" ]]; then
  green "guest checked in (session_id=$SESSION_ID)"
elif [[ "$SESSION_STATUS" == "open" ]]; then
  green "guest already checked in (idempotent)"
else
  red "check-in failed — $CHECKIN"
fi

# ── 10. Verify active session ──────────────────────────────────────────────────
step "10. Verify active session visibility"

# Guest sees own session
SESS=$(api GET "/sessions/guest/${GUEST_ID}" "$GUEST_TOKEN")
SESS_STATUS=$(echo "$SESS" | jq -r '.status // empty')
if [[ "$SESS_STATUS" == "open" ]]; then
  green "guest can view own session (status=open, spend=$(echo "$SESS" | jq -r '.total_spend_nc') NC)"
else
  red "guest session query failed — $SESS"
fi

# Manager view (nitecore) — list active sessions for venue
ACTIVE=$(api GET "/sessions/venues/${VENUE_ID}/active" "$NC_TOKEN")
SESS_COUNT=$(echo "$ACTIVE" | jq '.count // 0')
if [[ "$SESS_COUNT" -gt 0 ]]; then
  green "manager view: $SESS_COUNT active session(s) in venue"
else
  red "manager session list empty or failed — $ACTIVE"
fi

# ── 11. Bartender creates order ────────────────────────────────────────────────
step "11. Bartender creates order"
BAR_LOGIN=$(api POST /auth/login "" \
  "{\"email\":\"$BAR_EMAIL\",\"password\":\"BarPass123!\"}")
BAR_TOKEN=$(echo "$BAR_LOGIN" | jq -r '.access_token // empty')
if [[ -z "$BAR_TOKEN" ]]; then
  red "bartender login failed — $BAR_LOGIN"
else
  green "bartender login OK"
fi

IKEY="smoke:${TS}:order-1"
ORDER=$(api POST /orders/ "$BAR_TOKEN" \
  "{\"venue_id\":\"$VENUE_ID\",\"guest_session_id\":\"$SESSION_ID\",\"guest_user_id\":\"$GUEST_ID\",\"items\":[{\"item_id\":\"$ITEM_ID\",\"name\":\"Test Beer\",\"quantity\":1,\"price_nc\":8}],\"idempotency_key\":\"$IKEY\"}")
ORDER_ID=$(echo "$ORDER" | jq -r '.order_id // empty')
ORDER_STATUS=$(echo "$ORDER" | jq -r '.status // empty')
ORDER_TOTAL=$(echo "$ORDER" | jq -r '.total_nc // empty')
if [[ -n "$ORDER_ID" && "$ORDER_STATUS" == "pending" ]]; then
  green "order created (id=$ORDER_ID, status=$ORDER_STATUS, total=${ORDER_TOTAL} NC)"
else
  red "order creation failed — $ORDER"
fi

# ── 12. Finalize order (expect 402 — guest has 0 NC) ──────────────────────────
step "12. Finalize order (expect 402 insufficient balance)"
FINALIZE=$(api POST "/orders/${ORDER_ID}/finalize" "$BAR_TOKEN" \
  "{\"guest_user_id\":\"$GUEST_ID\"}")
HTTP_STATUS=$(echo "$FINALIZE" | jq -r '.error // "no_error"')
FINALIZE_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
  "${API_BASE}/orders/${ORDER_ID}/finalize" \
  -H "Authorization: Bearer $BAR_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"guest_user_id\":\"$GUEST_ID\"}")
if [[ "$FINALIZE_CODE" == "402" ]]; then
  green "finalize correctly returns 402 (insufficient balance — correct for 0 NC guest)"
elif [[ "$FINALIZE_CODE" == "409" ]]; then
  green "finalize returns 409 (already finalized from previous step — order flow is idempotent)"
else
  red "finalize unexpected HTTP $FINALIZE_CODE — $FINALIZE"
fi

# ── 13. Reporting revenue ──────────────────────────────────────────────────────
step "13. Reporting revenue endpoint"
TODAY=$(date -u +%Y-%m-%d)
REVENUE=$(api GET "/reporting/venues/${VENUE_ID}/revenue?from=2026-01-01&to=${TODAY}" "$NC_TOKEN")
if echo "$REVENUE" | jq -e '.venue_id' >/dev/null 2>&1; then
  green "reporting revenue OK (revenue_nc=$(echo "$REVENUE" | jq '.revenue_nc // 0'))"
else
  red "reporting revenue failed — $REVENUE"
fi

# ── Summary ────────────────────────────────────────────────────────────────────
echo ""
echo "=== Smoke test summary ==="
echo "  Venue ID:       $VENUE_ID"
echo "  Venue slug:     $VENUE_SLUG"
echo "  Guest ID:       $GUEST_ID  ($GUEST_EMAIL)"
echo "  Session ID:     ${SESSION_ID:-not_created}"
echo "  Order ID:       ${ORDER_ID:-not_created}"
echo "  Door staff ID:  ${DOOR_ID:-not_created}"
echo "  Bartender ID:   ${BAR_ID:-not_created}"
echo ""
echo "  NOTE: test data was NOT cleaned up. Delete via admin console if needed."
echo ""
if [[ $FAIL -eq 0 ]]; then
  echo -e "\033[32m✓ All smoke tests passed — stack is pilot-ready\033[0m"
  exit 0
else
  echo -e "\033[31m✗ Some smoke tests FAILED — review output above\033[0m"
  exit 1
fi
