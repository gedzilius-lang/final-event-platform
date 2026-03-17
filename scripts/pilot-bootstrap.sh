#!/usr/bin/env bash
# pilot-bootstrap.sh — Create the pilot venue, staff accounts, and catalog via API.
#
# Run ONCE after the stack is healthy and nitecore user exists.
# Uses the NiteOS API directly (requires gateway to be reachable).
# Idempotent where possible: re-running will create duplicate accounts unless you
# use the same emails — in that case auth/register returns 409, which is logged
# but not fatal.
#
# Requirements: curl, jq
#
# Usage:
#   API_BASE=https://api.peoplewelike.club \
#   NITECORE_EMAIL=nitecore@peoplewelike.club \
#   NITECORE_PASSWORD=<password> \
#   VENUE_NAME="People We Like" \
#   VENUE_SLUG=people-we-like \
#   VENUE_CITY=Zurich \
#   VENUE_CAPACITY=300 \
#   STAFF_PIN=<4+ digit PIN> \
#   ADMIN_EMAIL=admin@peoplewelike.club \
#   ADMIN_PASSWORD=<password> \
#   BARTENDER_EMAIL=bar@peoplewelike.club \
#   BARTENDER_PASSWORD=<password> \
#   DOOR_EMAIL=door@peoplewelike.club \
#   DOOR_PASSWORD=<password> \
#   bash scripts/pilot-bootstrap.sh
set -euo pipefail

API_BASE="${API_BASE:-https://api.peoplewelike.club}"
NITECORE_EMAIL="${NITECORE_EMAIL:-nitecore@peoplewelike.club}"
NITECORE_PASSWORD="${NITECORE_PASSWORD:-}"
VENUE_NAME="${VENUE_NAME:-People We Like}"
VENUE_SLUG="${VENUE_SLUG:-people-we-like}"
VENUE_CITY="${VENUE_CITY:-Zurich}"
VENUE_CAPACITY="${VENUE_CAPACITY:-300}"
VENUE_ADDRESS="${VENUE_ADDRESS:-}"
STAFF_PIN="${STAFF_PIN:-}"
ADMIN_EMAIL="${ADMIN_EMAIL:-}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
ADMIN_DISPLAY="${ADMIN_DISPLAY:-Venue Manager}"
BARTENDER_EMAIL="${BARTENDER_EMAIL:-}"
BARTENDER_PASSWORD="${BARTENDER_PASSWORD:-}"
BARTENDER_DISPLAY="${BARTENDER_DISPLAY:-Bar Staff}"
DOOR_EMAIL="${DOOR_EMAIL:-}"
DOOR_PASSWORD="${DOOR_PASSWORD:-}"
DOOR_DISPLAY="${DOOR_DISPLAY:-Door Staff}"
FAIL=0

if ! command -v jq &>/dev/null; then
  echo "ERROR: jq is required — apt-get install jq" >&2; exit 1
fi

# Validate required vars
for v in NITECORE_PASSWORD STAFF_PIN ADMIN_EMAIL ADMIN_PASSWORD BARTENDER_EMAIL BARTENDER_PASSWORD DOOR_EMAIL DOOR_PASSWORD; do
  if [[ -z "${!v:-}" ]]; then
    echo "ERROR: $v is required" >&2
    FAIL=1
  fi
done
if [[ $FAIL -eq 1 ]]; then
  echo ""
  echo "Usage: See script header for required env vars." >&2
  exit 1
fi

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

echo ""
echo "=== NiteOS Pilot Bootstrap ==="
echo "  API: $API_BASE"
echo "  Venue: $VENUE_NAME ($VENUE_SLUG)"
echo ""

# ── 1. Nitecore login ──────────────────────────────────────────────────────────
step "1. Nitecore login"
LOGIN=$(api POST /auth/login "" "{\"email\":\"$NITECORE_EMAIL\",\"password\":\"$NITECORE_PASSWORD\"}")
NC_TOKEN=$(echo "$LOGIN" | jq -r '.access_token // empty')
if [[ -z "$NC_TOKEN" ]]; then
  red "Nitecore login failed — $LOGIN"
  echo "Cannot continue without nitecore token." >&2
  exit 1
fi
green "Nitecore login OK"

# ── 2. Create venue ────────────────────────────────────────────────────────────
step "2. Create venue"
VENUE_BODY=$(jq -n \
  --arg name "$VENUE_NAME" \
  --arg slug "$VENUE_SLUG" \
  --arg city "$VENUE_CITY" \
  --arg addr "$VENUE_ADDRESS" \
  --argjson cap "$VENUE_CAPACITY" \
  --arg pin "$STAFF_PIN" \
  '{name:$name,slug:$slug,city:$city,address:$addr,capacity:$cap,staff_pin:$pin,timezone:"Europe/Zurich"}')
VENUE_RESP=$(api POST /catalog/venues "$NC_TOKEN" "$VENUE_BODY")
VENUE_ID=$(echo "$VENUE_RESP" | jq -r '.venue_id // empty')
if [[ -z "$VENUE_ID" ]]; then
  red "Create venue failed — $VENUE_RESP"
  echo "Cannot continue without venue_id." >&2
  exit 1
fi
green "Venue created: $VENUE_ID"

# ── 3. Seed catalog items ──────────────────────────────────────────────────────
step "3. Seed catalog items (1 NC = 1 CHF)"
seed_item() {
  local name="$1" cat="$2" price="$3" icon="$4" order="$5"
  local body resp
  body=$(jq -n --arg name "$name" --arg cat "$cat" --argjson price "$price" \
    --arg icon "$icon" --argjson ord "$order" \
    '{name:$name,category:$cat,price_nc:$price,icon:$icon,display_order:$ord}')
  resp=$(api POST "/catalog/venues/${VENUE_ID}/items" "$NC_TOKEN" "$body")
  local id
  id=$(echo "$resp" | jq -r '.item_id // empty')
  if [[ -n "$id" ]]; then
    green "  $name — ${price} NC = ${price} CHF (id=$id)"
  else
    red "  Failed to create item: $name — $resp"
  fi
}
seed_item "Entry"       "entry"  20 "🎟️" 1
seed_item "Draft Beer"  "drinks"  8 "🍺" 10
seed_item "Cocktail"    "drinks" 15 "🍹" 11
seed_item "Water"       "drinks"  4 "💧" 12
seed_item "Shot"        "drinks"  6 "🥃" 13
seed_item "Soft Drink"  "drinks"  4 "🥤" 14

# ── 4. Register staff accounts ─────────────────────────────────────────────────
step "4. Register and assign staff accounts"

register_and_assign() {
  local email="$1" password="$2" display="$3" role="$4"
  # Register (409 = already exists, not fatal)
  local reg_resp user_id
  reg_resp=$(api POST /auth/register "" \
    "{\"email\":\"$email\",\"password\":\"$password\",\"display_name\":\"$display\"}")
  user_id=$(echo "$reg_resp" | jq -r '.user_id // empty')

  if [[ -z "$user_id" ]]; then
    # Try login to get existing user_id
    local login_resp
    login_resp=$(api POST /auth/login "" "{\"email\":\"$email\",\"password\":\"$password\"}")
    user_id=$(echo "$login_resp" | jq -r '.user_id // empty')
    if [[ -z "$user_id" ]]; then
      red "  Cannot get user_id for $email — $reg_resp"
      return
    fi
    info "  $email already exists — assigning role"
  fi

  # Assign role + venue via nitecore
  local patch_resp
  patch_resp=$(api PATCH "/profiles/users/${user_id}/venue" "$NC_TOKEN" \
    "{\"venue_id\":\"$VENUE_ID\",\"role\":\"$role\"}")
  local assigned_role
  assigned_role=$(echo "$patch_resp" | jq -r '.role // empty')
  if [[ "$assigned_role" == "$role" ]]; then
    green "  $role: $email (id=$user_id)"
  else
    red "  Failed to assign $role to $email — $patch_resp"
  fi
}

register_and_assign "$ADMIN_EMAIL"      "$ADMIN_PASSWORD"      "$ADMIN_DISPLAY"      "venue_admin"
register_and_assign "$BARTENDER_EMAIL"  "$BARTENDER_PASSWORD"  "$BARTENDER_DISPLAY"  "bartender"
register_and_assign "$DOOR_EMAIL"       "$DOOR_PASSWORD"       "$DOOR_DISPLAY"       "door_staff"

# ── 5. Verify venue is listed ──────────────────────────────────────────────────
step "5. Verify venue listing"
VENUES=$(api GET /catalog/venues "$NC_TOKEN")
COUNT=$(echo "$VENUES" | jq '[.venues[] | select(.venue_id=="'"$VENUE_ID"'")] | length' 2>/dev/null || echo 0)
if [[ "$COUNT" -gt 0 ]]; then
  green "Venue appears in catalog listing"
else
  red "Venue not found in listing — $VENUES"
fi

# ── Summary ────────────────────────────────────────────────────────────────────
echo ""
echo "====================================="
echo "  Pilot Bootstrap Summary"
echo "====================================="
echo "  Venue:      $VENUE_NAME"
echo "  Venue ID:   $VENUE_ID"
echo "  Venue slug: $VENUE_SLUG"
echo "  Venue URL:  https://os.peoplewelike.club/venues/$VENUE_SLUG"
echo ""
echo "  Staff accounts:"
echo "    venue_admin  : $ADMIN_EMAIL"
echo "    bartender    : $BARTENDER_EMAIL"
echo "    door_staff   : $DOOR_EMAIL"
echo ""
echo "  Admin console: https://admin.peoplewelike.club"
echo "  Guest web:     https://os.peoplewelike.club"
echo "  API:           $API_BASE"
echo ""
if [[ $FAIL -eq 0 ]]; then
  echo -e "\033[32m✓ Pilot bootstrap complete — venue is ready for onboarding\033[0m"
  echo ""
  echo "  Next: read docs/PILOT_FLOW.md for operator walkthrough"
  echo ""
  exit 0
else
  echo -e "\033[31m✗ Bootstrap completed with errors — review output above\033[0m"
  echo ""
  exit 1
fi
