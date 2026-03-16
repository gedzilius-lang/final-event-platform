# DOMAIN_MODEL.md

Every entity in the NiteOS system, its fields, its relationships, and which service owns it. This is the canonical data model. Individual service schemas implement subsets of this model.

---

## Entity Relationship Overview

```
User ──────────────────── NiteTap (0..*)
 │                           │
 ├── VenueProfile (per venue) │
 │                            │
 └── VenueSession ────────────┘
          │
          ├── LedgerEvent (many)
          │
          └── Order (many)
                 │
                 ├── OrderItem (many)
                 └── LedgerEvent (1 reference)

Venue ─────────────────── CatalogItem (many)
  │                            │
  ├── Device (many)             └── Order.items[]
  ├── VenueSession (many)
  └── TicketProduct (many)
           │
           └── TicketIssuance ── LedgerEvent (1)

PaymentIntent ───────────── LedgerEvent (1 → topup_confirmed)

SyncFrame ────────────────── LedgerEvent[] (edge-sourced)
```

---

## Core Entities

### User

Owned by: **profiles service**
Table: `users`

```
user_id         uuid PRIMARY KEY DEFAULT gen_random_uuid()
email           text UNIQUE NOT NULL
password_hash   text NOT NULL
display_name    text
role            text NOT NULL DEFAULT 'guest'
                  -- guest | venue_admin | bartender | door_staff | nitecore
global_xp       integer NOT NULL DEFAULT 0
global_level    integer NOT NULL DEFAULT 1
created_at      timestamptz NOT NULL DEFAULT now()
updated_at      timestamptz NOT NULL DEFAULT now()
```

Notes:
- No wallet balance field. Ever.
- `role` here is the platform role. Operational roles (bartender, door_staff) are set per-device in the devices service, not on the user record.
- A user can be both a guest on the platform and a venue_admin for a specific venue. The venue_id association lives in VenueProfile, not here.
- XP and level are global (network-wide). Venue-local XP lives in VenueProfile.

---

### VenueProfile

Owned by: **profiles service**
Table: `venue_profiles`

```
profile_id      uuid PRIMARY KEY DEFAULT gen_random_uuid()
user_id         uuid NOT NULL REFERENCES users(user_id)
venue_id        uuid NOT NULL REFERENCES venues(venue_id)
local_xp        integer NOT NULL DEFAULT 0
local_level     integer NOT NULL DEFAULT 1
first_visit_at  timestamptz
last_visit_at   timestamptz
visit_count     integer NOT NULL DEFAULT 0
preferences     jsonb DEFAULT '{}'
UNIQUE (user_id, venue_id)
```

Notes:
- Created on first check-in at a venue.
- `preferences` holds venue-specific settings (notification opt-in, favorite items, etc.).

---

### NiteTap

Owned by: **profiles service**
Table: `nitetaps`

```
tap_id          uuid PRIMARY KEY DEFAULT gen_random_uuid()
nfc_uid         text UNIQUE NOT NULL     -- raw NFC tag UID (hex string)
user_id         uuid REFERENCES users(user_id)  -- NULL = anonymous token
venue_id        uuid REFERENCES venues(venue_id) -- NULL = not venue-scoped
is_anonymous    boolean NOT NULL DEFAULT true
issued_at       timestamptz NOT NULL DEFAULT now()
registered_at   timestamptz                -- when user linked it to an account
revoked_at      timestamptz                -- NULL = active
status          text NOT NULL DEFAULT 'active'  -- active | revoked | replaced
metadata        jsonb DEFAULT '{}'
```

Notes:
- Anonymous NiteTaps exist without a user_id. They carry a wallet balance in the ledger under a synthetic user_id generated at issuance.
- When a guest registers after using an anonymous tap, the synthetic user_id is merged into the real user_id via a ledger migration event.
- Replay attack protection: each tap interaction includes an incrementing counter validated by the edge node.

---

### Venue

Owned by: **catalog service**
Table: `venues`

```
venue_id        uuid PRIMARY KEY DEFAULT gen_random_uuid()
name            text NOT NULL
slug            text UNIQUE NOT NULL
city            text NOT NULL DEFAULT 'Zurich'
address         text
capacity        integer NOT NULL DEFAULT 200
staff_pin       text NOT NULL              -- hashed PIN for venue PIN login
timezone        text NOT NULL DEFAULT 'Europe/Zurich'
theme           jsonb DEFAULT '{}'         -- brand_color, logo_url, font
stripe_account  text                       -- Stripe Connect account ID (Phase 2)
is_active       boolean NOT NULL DEFAULT true
created_at      timestamptz NOT NULL DEFAULT now()
```

Notes:
- `theme` drives Venue Mode UI re-skin in guest-web.
- `staff_pin` is bcrypt-hashed. Used for PIN-based terminal login.
- `stripe_account` is NULL until Phase 2 Stripe Connect activation.

---

### Device

Owned by: **devices service**
Table: `devices`

```
device_id       uuid PRIMARY KEY DEFAULT gen_random_uuid()
venue_id        uuid NOT NULL REFERENCES venues(venue_id)
device_role     text NOT NULL    -- kiosk | terminal | master
device_name     text             -- human label: "Bar 1 Kiosk", "Front Door"
public_key      text NOT NULL    -- PEM public key for device credential auth
status          text NOT NULL DEFAULT 'pending'
                  -- pending | active | revoked
enrolled_at     timestamptz
last_heartbeat  timestamptz
last_seen_ip    text
firmware_ver    text
metadata        jsonb DEFAULT '{}'
```

Notes:
- Devices generate their own keypair on first boot. Public key submitted during enrollment.
- Auth service holds the device JWT; devices service holds the enrollment record.
- Heartbeat is written by the edge service on behalf of all connected terminals.

---

### LedgerEvent

Owned by: **ledger service**
Table: `ledger_events`

```
event_id            uuid PRIMARY KEY DEFAULT gen_random_uuid()
event_type          text NOT NULL
  -- topup_pending | topup_confirmed | order_paid | refund_created
  -- bonus_credit | ticket_purchase | venue_checkin | session_closed
  -- merge_anonymous (internal)
user_id             uuid NOT NULL
venue_id            uuid
device_id           uuid
amount_nc           integer NOT NULL
  -- positive = credit to user, negative = debit from user
amount_chf          numeric(10,2)
  -- original fiat value if this event was triggered by a payment
reference_id        uuid
  -- order_id | payment_intent_id | ticket_issuance_id | session_id
idempotency_key     text UNIQUE NOT NULL
  -- prevents duplicate events; format: {source}:{reference_id}:{event_type}
occurred_at         timestamptz NOT NULL DEFAULT now()
synced_from         text DEFAULT 'cloud'
  -- 'cloud' | 'edge:{venue_id}' (set by sync service on edge-sourced events)
written_by          text NOT NULL
  -- service name that wrote this event: 'payments' | 'orders' | 'sessions' etc.
metadata            jsonb DEFAULT '{}'
```

Constraints enforced in application code (not just DB):
- INSERT only. No UPDATE statement touches this table. No DELETE statement touches this table.
- `amount_nc` cannot be zero.
- `idempotency_key` must be supplied by the caller.
- Only the authorised calling service may write specific event_types (enforced via service identity headers).

Balance projection query (wallet service):
```sql
SELECT COALESCE(SUM(amount_nc), 0) AS balance_nc
FROM ledger_events
WHERE user_id = $1
  AND event_type NOT IN ('topup_pending', 'venue_checkin', 'session_closed')
  AND (venue_id = $2 OR venue_id IS NULL)
```

---

### PaymentIntent

Owned by: **payments service**
Table: `payment_intents`

```
intent_id       uuid PRIMARY KEY DEFAULT gen_random_uuid()
user_id         uuid NOT NULL
venue_id        uuid NOT NULL
provider        text NOT NULL         -- 'twint' | 'stripe'
provider_ref    text                  -- provider's own reference ID
amount_chf      numeric(10,2) NOT NULL
status          text NOT NULL DEFAULT 'pending'
  -- pending | confirmed | captured | refunded | failed | expired
created_at      timestamptz NOT NULL DEFAULT now()
confirmed_at    timestamptz
webhook_payload jsonb                 -- raw verified webhook body
idempotency_key text UNIQUE NOT NULL
```

Notes:
- This is the only place that records the fiat transaction.
- On `status = captured`, payments service writes `topup_confirmed` to ledger with `reference_id = intent_id`.
- On refund, payments service writes `refund_created` to ledger and triggers provider refund API.

---

### TicketProduct

Owned by: **ticketing service**
Table: `ticket_products`

```
product_id      uuid PRIMARY KEY DEFAULT gen_random_uuid()
venue_id        uuid NOT NULL
event_id        uuid                  -- links to event listing (optional)
title           text NOT NULL
description     text
price_chf       numeric(10,2) NOT NULL
nc_included     integer NOT NULL DEFAULT 0  -- bonus NC credited on purchase
capacity        integer               -- NULL = unlimited
sold_count      integer NOT NULL DEFAULT 0
status          text NOT NULL DEFAULT 'draft'  -- draft | active | sold_out | archived
valid_from      timestamptz
valid_until     timestamptz
created_at      timestamptz NOT NULL DEFAULT now()
```

---

### TicketIssuance

Owned by: **ticketing service**
Table: `ticket_issuances`

```
issuance_id     uuid PRIMARY KEY DEFAULT gen_random_uuid()
product_id      uuid NOT NULL REFERENCES ticket_products(product_id)
user_id         uuid NOT NULL
venue_id        uuid NOT NULL
qr_token        text UNIQUE NOT NULL   -- HMAC-signed, single-use scan token
status          text NOT NULL DEFAULT 'valid'
  -- valid | used | refunded | expired
ledger_event_id uuid NOT NULL          -- ticket_purchase event that created this
issued_at       timestamptz NOT NULL DEFAULT now()
used_at         timestamptz
```

Notes:
- `qr_token` is generated as: `HMAC-SHA256(secret, issuance_id + user_id + product_id)` → base64url encoded.
- On check-in scan: sessions service calls ticketing service to validate and mark used.

---

### Order

Owned by: **orders service**
Table: `orders`

```
order_id            uuid PRIMARY KEY DEFAULT gen_random_uuid()
venue_id            uuid NOT NULL
device_id           uuid NOT NULL      -- which kiosk processed this
staff_user_id       uuid               -- which bartender (if staff-scoped)
guest_session_id    uuid NOT NULL REFERENCES venue_sessions(session_id)
total_nc            integer NOT NULL
status              text NOT NULL DEFAULT 'pending'
  -- pending | paid | voided | refunded
ledger_event_id     uuid               -- order_paid event (set on finalization)
idempotency_key     text UNIQUE NOT NULL
created_at          timestamptz NOT NULL DEFAULT now()
finalized_at        timestamptz
```

---

### OrderItem

Owned by: **orders service**
Table: `order_items`

```
item_id         uuid PRIMARY KEY DEFAULT gen_random_uuid()
order_id        uuid NOT NULL REFERENCES orders(order_id)
catalog_item_id uuid NOT NULL
name            text NOT NULL          -- snapshot of item name at time of order
price_nc        integer NOT NULL       -- snapshot of price at time of order
quantity        integer NOT NULL DEFAULT 1
```

Notes:
- Name and price are snapshot-copied at order time. Changing the catalog after an order does not retroactively change order records.

---

### CatalogItem

Owned by: **catalog service**
Table: `catalog_items`

```
item_id         uuid PRIMARY KEY DEFAULT gen_random_uuid()
venue_id        uuid NOT NULL
name            text NOT NULL
category        text NOT NULL
price_nc        integer NOT NULL
icon            text                   -- emoji or icon identifier
stock_qty       integer                -- NULL = not tracked
low_threshold   integer DEFAULT 5
is_active       boolean NOT NULL DEFAULT true
display_order   integer DEFAULT 0
happy_hour_price_nc integer            -- price during happy hour (NULL = no override)
created_at      timestamptz NOT NULL DEFAULT now()
updated_at      timestamptz NOT NULL DEFAULT now()
```

---

### HappyHourRule

Owned by: **catalog service**
Table: `happy_hour_rules`

```
rule_id         uuid PRIMARY KEY DEFAULT gen_random_uuid()
venue_id        uuid NOT NULL
name            text NOT NULL
starts_at       time NOT NULL          -- e.g. 20:00
ends_at         time NOT NULL          -- e.g. 22:00
days_of_week    integer[]              -- 0=Sun, 1=Mon, ... 6=Sat (NULL = every day)
price_modifier  numeric(4,2)           -- e.g. 0.80 = 20% off
is_active       boolean NOT NULL DEFAULT true
```

---

### VenueSession

Owned by: **sessions service**
Table: `venue_sessions`

```
session_id      uuid PRIMARY KEY DEFAULT gen_random_uuid()
user_id         uuid NOT NULL
venue_id        uuid NOT NULL
nitetap_uid     text                   -- NFC UID used to open this session (may be anonymous)
ticket_used     uuid                   -- ticket_issuance_id if entry was via ticket
opened_at       timestamptz NOT NULL DEFAULT now()
closed_at       timestamptz
total_spend_nc  integer NOT NULL DEFAULT 0  -- running total, updated on each order
checkin_device  uuid                   -- which NiteTerminal scanned them in
status          text NOT NULL DEFAULT 'open'  -- open | closed
```

Notes:
- `total_spend_nc` is a running aggregate updated by the orders service for display purposes. The ledger is still the truth; this is a convenience field for fast dashboard reads.

---

### SyncFrame

Owned by: **sync service**
Table: `sync_frames`

```
frame_id        uuid PRIMARY KEY DEFAULT gen_random_uuid()
venue_id        uuid NOT NULL
device_id       uuid NOT NULL          -- Master Tablet that submitted this frame
event_count     integer NOT NULL
event_id_range  text NOT NULL          -- "{first_event_id}:{last_event_id}"
checksum        text NOT NULL          -- SHA256 of serialized events
events          jsonb NOT NULL         -- array of LedgerEvent-shaped objects
submitted_at    timestamptz NOT NULL DEFAULT now()
processed_at    timestamptz
status          text NOT NULL DEFAULT 'received'
  -- received | processing | processed | failed
failure_reason  text
```

---

### Event (Public Event Listing)

Owned by: **catalog service** (venue event listings, not audit events)
Table: `events`

```
event_id        uuid PRIMARY KEY DEFAULT gen_random_uuid()
venue_id        uuid NOT NULL
title           text NOT NULL
starts_at       timestamptz NOT NULL
ends_at         timestamptz
description     text
genre           text
image_url       text
is_public       boolean NOT NULL DEFAULT true
created_at      timestamptz NOT NULL DEFAULT now()
```

Notes:
- These are nightlife event listings shown in the guest-web event feed.
- Not to be confused with ledger events or audit events. The word "event" is overloaded — in this model, unqualified "event" always means a public venue event. Internal system events are always prefixed: "ledger event", "sync event", "audit event".

---

## Entity Ownership Summary

| Entity | Owning Service | Postgres Schema |
|--------|---------------|-----------------|
| User | profiles | `profiles` |
| VenueProfile | profiles | `profiles` |
| NiteTap | profiles | `profiles` |
| Venue | catalog | `catalog` |
| Device | devices | `devices` |
| LedgerEvent | ledger | `ledger` |
| PaymentIntent | payments | `payments` |
| TicketProduct | ticketing | `ticketing` |
| TicketIssuance | ticketing | `ticketing` |
| Order | orders | `orders` |
| OrderItem | orders | `orders` |
| CatalogItem | catalog | `catalog` |
| HappyHourRule | catalog | `catalog` |
| VenueSession | sessions | `sessions` |
| SyncFrame | sync | `sync` |
| Event (listing) | catalog | `catalog` |

Each service connects to the same Postgres instance but operates only within its own schema. A service must never issue a query against another service's schema directly — it must call the owning service's API.

---

## Cross-Service ID References (Read-Only)

Services frequently need to reference entities owned by other services. The rule: store the foreign UUID, never join across schemas.

| Service | Stores | Why |
|---------|--------|-----|
| orders | `venue_id`, `device_id`, `guest_session_id` | Reference only — never joined, validated at write time |
| ledger | `user_id`, `venue_id`, `device_id`, `reference_id` | Audit trail — cross-service IDs stored for traceability |
| sessions | `user_id`, `venue_id`, `ticket_used` | Reference only |
| ticketing | `venue_id`, `user_id`, `ledger_event_id` | Reference only |
| sync | `venue_id`, `device_id` | Frame attribution |

When a service needs to validate a cross-service ID (e.g., "does this venue_id actually exist?"), it calls the owning service's API at write time and caches the result. It does NOT query the other schema.
