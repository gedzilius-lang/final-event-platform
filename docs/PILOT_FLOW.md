# NiteOS Pilot Flow

Exact walkthrough for an operator running the first live session.
This assumes the stack is deployed, healthy, and bootstrap is complete.

---

## Prerequisites

| Item | Where | Status |
|------|-------|--------|
| Stack deployed + healthy | `bash scripts/healthcheck-cloud.sh` | Must pass |
| Nitecore account exists | `bash scripts/bootstrap-nitecore.sh` | Must be done |
| Pilot venue + staff created | `bash scripts/pilot-bootstrap.sh` | Must be done |
| Admin console accessible | `https://admin.peoplewelike.club` | Verify in browser |
| Guest web accessible | `https://os.peoplewelike.club` | Verify in browser |

---

## Accounts needed

| Role | Email | Logs in at |
|------|-------|-----------|
| Nitecore (super-admin) | as configured | admin console |
| Venue admin / manager | as configured | admin console + guest web /staff |
| Bartender | as configured | guest web /staff/bar |
| Door staff | as configured | guest web /staff/door |
| Test guest | new account via /register | guest web |

---

## Step 1 — Guest registers and loads wallet

**Where:** Guest opens `https://os.peoplewelike.club` on their phone.

1. Tap **Create account** → enter email, password, display name → submit
2. On home screen, tap **Wallet** in bottom nav
3. Tap **Top Up** → select amount (e.g., 50 CHF = 50 NC) → complete Stripe payment
4. After redirect, wallet shows **50 NC** balance
5. Optionally: tap **Profile** to see Level 1, XP bar, no history yet

**What the system does:**
- Auth service creates user (role=guest)
- Ledger writes `topup_pending` on payment initiation
- Stripe webhook fires → ledger writes `topup_confirmed`
- Balance becomes spendable (topup_confirmed is counted, topup_pending is not)

---

## Step 2 — Door staff logs in and checks in the guest

**Where:** Door staff opens `https://os.peoplewelike.club` → `/login` on venue tablet/phone.

1. Log in with `door_staff` credentials
2. Automatically redirected to `/staff/door` (door staff home)
3. Active session count shows current headcount
4. To check in guest:
   - Enter guest **email** in the lookup field → tap **Look up**
   - Guest card appears with display name and role
   - Optionally enter NiteTap UID if guest has a wristband
   - Tap **Check In** to confirm
5. Session counter increments

**What the system does:**
- `POST /sessions/checkin` creates a `venue_session` with status=open
- Idempotent: checking in the same guest twice returns the existing session
- Sessions service fires fire-and-forget calls:
  - Ledger: writes `venue_checkin` event (informational, excluded from balance)
  - Profiles: upserts `venue_profile` (visit count +1)
  - Profiles: awards 10 XP global + 10 XP local (non-fatal)

---

## Step 3 — Guest orders at the bar

**Where:** Bartender opens `https://os.peoplewelike.club` → `/login` on POS tablet.

1. Log in with `bartender` credentials → redirected to `/staff/bar`
2. Menu grid shows catalog items with prices in NC
3. To take an order:
   - Tap items to add to cart (badge shows quantity)
   - **Identify guest** (optional but required for wallet charge):
     - Enter **NiteTap UID** if guest has a wristband → tap Identify
     - OR skip (anonymous order — no wallet charge)
   - Tap **Charge Guest** (or **Mark as Paid** for anonymous)
4. On success: order confirmation, cart clears

**What the system does:**
- `POST /orders/` creates order with status=pending
- `POST /orders/{id}/finalize` with `guest_user_id`:
  - Checks balance against `GET /ledger/balance/{userId}/venue/{venueId}`
  - If sufficient: writes `order_paid` ledger event (negative amount)
  - Marks order status=paid
  - Fire-and-forget: increments `total_spend_nc` on the session
  - Fire-and-forget: awards XP (1 NC spent = 1 XP global + 1 XP local)
  - If insufficient: returns **402 Payment Required** — tell guest to top up

---

## Step 4 — Guest checks their wallet and session

**Where:** Guest web on guest's phone.

1. Home screen shows updated balance (reduced by order amount)
2. Active session banner shows "You're checked in!" with session spend
3. Tap **Profile** → XP bar updated, activity log shows the purchase

---

## Step 5 — Manager views live operations

**Where:** Venue admin opens `https://os.peoplewelike.club` → `/login` on manager device.

1. Log in with `venue_admin` credentials → redirected to `/staff/manager`
2. Dashboard shows:
   - Live session count (active check-ins)
   - Today's revenue in NC
   - Quick links to door/bar/security surfaces
3. Full admin view available at `https://admin.peoplewelike.club`:
   - Sessions tab: all active sessions with spend
   - Reports tab: revenue breakdown by date
   - Catalog tab: edit items, prices, toggle active/inactive

---

## Step 6 — Guest checks out (end of night)

**Where:** Door staff OR guest themselves.

**Guest self-checkout:**
- Guest web → `/session` → tap **Check Out**
- Session closes, `session_closed` event written to ledger (informational)

**Door staff checkout:**
- `/staff/door` → find guest by email → checkout button on active session card

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Door staff can't log in | Role not assigned | Nitecore: `PATCH /profiles/users/{id}/venue` with role=door_staff |
| Bartender sees empty menu | Catalog items inactive or wrong venue | Admin console → Catalog → toggle active |
| Guest charge fails with 402 | Insufficient balance | Guest must top up wallet |
| Session not showing for manager | Guest not checked in | Door staff must POST /sessions/checkin first |
| Balance not updating after topup | Stripe webhook not reaching stack | Check `docker compose logs payments`; verify webhook URL in Stripe dashboard |
| XP not updating | Profiles service call failed | Non-fatal — XP updates are fire-and-forget; check `docker compose logs profiles` |
| Radio not playing | Stream URL incorrect or stream offline | Check `NEXT_PUBLIC_RADIO_STREAM_URL` in cloud.env |

---

## API reference (quick)

All via `https://api.peoplewelike.club`. JWT Bearer token from login required.

| Action | Method | Path |
|--------|--------|------|
| Login | POST | `/auth/login` |
| Register | POST | `/auth/register` |
| Wallet balance | GET | `/wallet/{userId}` |
| Wallet history | GET | `/wallet/{userId}/history` |
| User profile (XP, level) | GET | `/profiles/users/{userId}` |
| Active session for user | GET | `/sessions/guest/{userId}` |
| Check in guest | POST | `/sessions/checkin` |
| Active sessions for venue | GET | `/sessions/venues/{venueId}/active` |
| Venue catalog items | GET | `/catalog/venues/{venueId}/items` |
| Create order | POST | `/orders/` |
| Finalize order | POST | `/orders/{orderId}/finalize` |
| Reporting revenue | GET | `/reporting/venues/{venueId}/revenue?from=YYYY-MM-DD&to=YYYY-MM-DD` |
| Lookup guest by email | GET | `/profiles/users/by-email/{email}` |
| Lookup guest by NiteTap | GET | `/profiles/users/by-nfc-uid/{uid}` |

---

## NC / CHF reference

| NC | CHF | Common use |
|----|-----|-----------|
| 4 | 4.00 | Water |
| 6 | 6.00 | Shot |
| 8 | 8.00 | Draft beer |
| 15 | 15.00 | Cocktail |
| 20 | 20.00 | Entry |
| 50 | 50.00 | Typical min topup |

**1 NC = 1 CHF — fixed peg. No conversion factor.**
