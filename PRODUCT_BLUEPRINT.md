# PRODUCT_BLUEPRINT.md

The definitive specification of what NiteOS is, what it contains, and how it is structured. This document resolves all contradictions and supersedes partial definitions in earlier notes.

---

## What NiteOS Is

NiteOS is a **white-label venue operating system** for the permanent nightlife industry — clubs, lounges, and bars. It is not a festival app. It is not a generic POS. It is not a loyalty bolt-on. It is the complete operational layer that a venue runs on: entry, identity, spend, staff, reporting, and cash flow.

The original concept (NEA — NFC Event Assistant) established the core thesis in 2023: use NFC to replace cash at events, capture spending data, and give operators real-time visibility. NiteOS is the evolved, production-grade version of that thesis — extended from single events to permanent venues, from mobile-first to offline-capable infrastructure, and from feature list to deployable operating system.

The system has three layers that together form one product:

1. **NiteOS Cloud Core** — the durable platform (identity, ledger, payments, reporting)
2. **NiteOS Edge** — the venue-local node (authoritative during live operations, offline-capable)
3. **NiteOS Terminals** — the hardware layer (bartender kiosk, door scanner, manager tablet)

People We Like is the launch brand and cultural vehicle. The radio and marketplace operate under the same brand but as separate systems. NiteOS is the infrastructure. People We Like is the identity.

---

## The Product Surfaces

### 1. Guest App (os.peoplewelike.club)
Browser-first in MVP. One app, two modes.

**Global Mode** (outside a venue):
- Discover upcoming events and venues
- Buy tickets with wallet credit or card
- NiteMarket: purchase access bundles, drink credit packs, skip-the-line passes
- Wallet: view NiteCoin balance, top up via TWINT or Stripe, see history
- Profile: level, achievements, visit history, friends
- Radio: embedded People We Like Radio player

**Venue Mode** (after check-in tap or QR scan):
- App re-skins to venue branding (color, logo, typography)
- Venue-level identity: "Club Noir Agent — Level 3"
- Live tab: view current session spend, remaining balance
- Friends radar: see which friends are also checked in
- Self-checkout (if venue enables): order via app
- Happy Hour timer, secret menu unlocks based on local level
- Exit: ends session, returns to Global Mode

Mode switch trigger: user taps NiteTap on NiteKiosk at door (check-in), or scans venue QR code. Mode ends when door staff manually closes session or user self-exits.

### 2. Bartender Terminal — NiteKiosk
Android phone in kiosk mode (Device Owner enforced). Mounted at bar. Ruggedized.

- High-contrast dark POS UI (designed for low-light, wet hands)
- Menu by category with large touch targets
- Tap-to-collect: staff presents NiteKiosk, guest taps NiteTap → NiteCoin deducted instantly
- QR scan as fallback identity method
- Offline-first: all transactions processed locally on edge node via Wi-Fi LAN; cloud fallback when LAN down
- Live stock: each transaction decrements inventory; "86" alert when item low
- No refund above threshold without Venue Admin approval
- No access to any admin, reporting, or other venue data

### 3. Door Terminal — NiteTerminal
Android phone in kiosk mode. Door staff / security role.

- Multi-format scan: QR ticket, NFC NiteTap, physical ID (visual only, no biometric)
- Check-in: validates ticket, opens venue session, VIP haptic alert for high-level users
- Capacity counter: live in/out tally synced across all door staff devices
- Occupancy dashboard: real-time % of venue capacity
- Walk-in onboarding: scan blank NiteTap card → create anonymous session instantly (cash-in loads NiteCoins)
- No access to financial data, menu, or admin functions

### 4. Manager Tablet — Master Tablet
Android tablet (Samsung Galaxy Tab Active 5 or equivalent, rugged). Runs both edge service and admin UI.

- Live Pulse Dashboard: NiteCoin spend tonight, top items, staff activity
- Heatmap: bar/area breakdown of revenue
- Menu management: update prices, add/hide items, configure Happy Hour timers
- Staff management: create/revoke access, set roles
- Device management: enroll NiteKiosk and NiteTerminal devices
- Push broadcast: send notification to all currently checked-in guests
- End-of-night close: finalize session, export summary, trigger sync flush
- Override functions: manual balance adjustment, void correction, break-glass audit

### 5. Admin Console (admin.peoplewelike.club)
Next.js web. Venue Admin and Nitecore HQ operators.

**Venue Admin view:**
- Same as Manager Tablet but in browser
- Post-event reports and CSV exports
- Catalog and pricing management
- Staff roles and device enrollment
- Billing and hardware status

**Nitecore HQ view (NiteDash / NiteCore):**
- Network overview: all venues, node health, sync status
- The Mint: total NiteCoin in circulation, top-up rates, breakage accumulation
- Revenue view: real-time transaction fees, SaaS billing
- Venue onboarding: create new venue tenants, generate API keys
- Fraud detection: flagged staff with high void/refund rates
- Global analytics: category trends across the network

### 6. People We Like Radio (radio.peoplewelike.club)
Separate deployment, separate VPS. Embedded as player in Global Mode guest app.

- 24/7 AutoDJ with Zurich timezone daypart scheduling
- Live RTMP ingest (OBS/Blackmagic) with seamless switching
- HLS output via relay daemon (monotonic segment IDs)
- Now playing and status API endpoints
- Not part of NiteOS operational stack. Cannot block venue operations.

### 7. People We Like Market (market.peoplewelike.club)
Separate deployment. Brand commerce layer.

- Curated multi-vendor marketplace for physical goods (handmade, limited-edition)
- Vendor management: applications, products, editorial drops, taste graph
- Operates independently of NiteOS. Not part of venue operations.
- Future: shared guest identity so the same account works across Market and NiteOS

---

## NiteCoin — The Economic Engine

NiteCoin (NC) is the internal unit of account for the entire network.

**Peg:** 1 NC = 1 CHF, fixed. No exchange rate risk.

**Legal classification:** Multi-Purpose Voucher (MPV). Redeemable only within the NiteOS partner network. Non-transferable outside the ecosystem. No cash withdrawals.

**Top-up model:** A guest pays 100 CHF and receives 100 NC. Venues may configure a bonus on top (e.g., +10 NC on top-ups over 50 CHF). The bonus NC is absorbed by the venue as a marketing expense (equivalent to a discount), not by Nitecore. This keeps the 1:1 peg clean.

**Spend model:** NiteCoins are deducted from the guest's wallet when a NiteKiosk processes an order. The deduction writes a ledger event. Balances are always projections from the immutable ledger — never stored as a mutable number.

**Breakage:** NiteCoins expire after 365 days of inactivity. Unspent balances at expiry are split contractually between Nitecore and the Venue Partner (agreed % in venue contract). Breakage is 100% margin revenue for both parties.

**Settlement flow (Phase 2 and beyond):** When a guest tops up 100 CHF via Stripe Connect:
- 95 CHF routes to the Venue's connected Stripe account (held in reserve pool)
- 5 CHF routes to Nitecore account (transaction fee)
- NC is minted in the ledger after verified payment callback
- Venue funds are released on agreed schedule (daily/weekly rolling basis)

**Ledger invariant (hard rule):** No NC is ever minted until the payment provider sends a verified, signature-checked webhook callback. This is enforced at the ledger service level. It cannot be overridden by any UI or API call without Nitecore superadmin authority.

---

## NiteTap — The Hardware Key

NiteTap is a physical NFC wearable (bracelet, ring, or card) with an embedded encrypted NFC chip. It is an **authentication key**, not a wallet.

**What it stores:** A single UID. Nothing else. No balance, no personal data.
**Where data lives:** Server-side only. If a NiteTap is cloned, stolen, or lost, the account can be re-keyed to a new physical tag instantly. The compromised UID is invalidated in the devices service.

**Replay attack protection:** Every NiteTap interaction generates a unique signature based on an incrementing counter. The edge node rejects any signature already seen.

**Anonymous by default:** A NiteTap can be issued as a "One-Night Token" at the door without any guest registration. The guest taps, gets an anonymous session and a wallet with any cash-loaded NC. Registration (email) is optional and incentivized (free shot, bonus NC, loyalty access). This maximizes conversion while minimizing friction at entry.

**Hardware specification (to confirm):** Compatible NFC chip type, frequency, read range, and OS compatibility across Android Kotlin SDK must be locked before hardware procurement. Current implementation assumption: ISO 14443-A or ISO 15693 passive NFC tags at 13.56 MHz. Minimum: NTAG213 or equivalent.

**Cost target:** < 1 CHF per unit at scale. Replacement cost is negligible. Lost or damaged NiteTap = scan a new tag, re-associate to account.

---

## The Offline Architecture (Non-Negotiable)

If the internet dies at 11 PM on a Saturday, the venue continues operating. This is the primary architectural constraint. It is also the primary competitive moat.

**Edge node** runs on the Master Tablet (later: optional NiteBox hardware unit). It provides:
- SQLite hot ledger: local append-only event store
- Local order capture: bartender POS works at full speed on LAN only
- Local check-in state: door staff scans work offline
- Local device coordination: all terminals communicate with edge, not cloud
- Sync queue: events buffered during cloud outage
- Sync agent: flushes buffered events to cloud on connectivity restoration

**Terminal connectivity order:**
1. Terminal → Edge Node over venue LAN (preferred, lowest latency)
2. Terminal → Cloud direct (fallback only when edge unreachable)

**Sync protocol properties:**
- Idempotent: the same event submitted twice produces the same state once
- Append-only: sync frames carry ledger events, never mutations
- Monotonic: events have sequence numbers; cloud handles out-of-order arrival
- Duplicate-tolerant: cloud deduplicates by event ID
- Auditable: every sync frame carries venue ID, device ID, timestamp, checksum

**Edge authority:** The edge node is the source of truth during live venue operations. The cloud is the durable aggregation and control plane. These are not interchangeable roles.

---

## Data Architecture

### Cloud (Postgres + Redis)
- `ledger_events` — append-only, immutable, the most important table in the system
- `users` — email, auth, global identity
- `venues` — venue configuration, partner settings, device registry
- `wallet_state` — projections from ledger (read-through cache, not source of truth)
- `tickets` — ticket products, inventory, issuance, redemption
- `orders` — POS order records, referencing ledger event IDs
- `catalog` — venue menu items, prices, availability
- `devices` — enrollment, per-device keys, heartbeat, role
- `sessions` — Redis: JWT session store with revocation list
- `cache` — Redis: rate limits, wallet balance cache, queue coordination

### Edge (SQLite)
- `edge_ledger` — local hot ledger (subset of cloud ledger, venue-scoped)
- `edge_orders` — pending orders awaiting sync
- `edge_checkins` — check-in state for current session
- `sync_queue` — unsynced events awaiting connectivity
- `device_state` — local device registry and status
- `catalog_cache` — synced copy of venue catalog

### Key Data Rules
- Wallet balance is never stored. It is always computed from ledger projection.
- Orders always reference the ledger event ID that consumed the NC.
- Refunds are compensating events. They do not delete or modify existing events.
- No NC is minted without a verified payment callback.
- Edge ledger events that fail cloud sync are never discarded — they are queued indefinitely.

---

## Technology Stack (Locked)

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Cloud backend | Go 1.22+, static binaries | Operational reliability, small footprint, concurrency |
| Web frontends | TypeScript + React + Next.js | Consistent, team-familiar, SSR for public pages |
| Android terminals | Kotlin, Device Owner API | Kiosk enforcement, reliable NFC, no web runtime |
| Cloud database | PostgreSQL 16 | Durable, transactional, mature |
| Cache / sessions | Redis | Session revocation, rate limiting, wallet cache |
| Edge database | SQLite | Embedded, offline-capable, zero-config |
| Async messaging | NATS (optional in pilot) | Clean service decoupling, useful at scale |
| Reverse proxy | Traefik with Cloudflare DNS-01 TLS | Unified TLS, dynamic routing |
| Observability | Grafana (first), then Loki + Prometheus | Alerting before metrics collection |
| Payments | TWINT (primary), Stripe (secondary) | Swiss market requirement |
| Hardware | Samsung Galaxy Tab Active 5 (tablet), Android phones (terminals) | Ruggedized, device owner support |
| Networking | Ubiquiti UniFi (dedicated VLAN per venue) | Isolates NiteOS from guest Wi-Fi congestion |

---

## System Boundaries

### What NiteOS owns
- Guest identity across venue interactions
- The NiteCoin ledger and wallet
- Ticket issuance and check-in
- POS at the bar and door
- Device enrollment and kiosk enforcement
- Venue configuration and reporting
- Payment flows (TWINT/Stripe → NC minting)

### What NiteOS does not own
- Guest Wi-Fi (venue's responsibility; NiteOS uses dedicated VLAN)
- Alcohol licensing or venue compliance
- Physical security beyond digital access control
- Accounting, payroll, or vendor procurement
- Physical stock management (NiteOS tracks digital inventory; physical reordering is the venue's responsibility)

### Separation of properties
- **Radio** (radio.peoplewelike.club): separate VPS, separate codebase, separate ops. Linked in guest app as embedded player. Has no access to NiteOS operational data.
- **Market** (market.peoplewelike.club): separate VPS, separate codebase, separate ops. Shares brand identity. Future: shared user account. Has no operational role in venue management.

---

## Deployment Model

### Infrastructure
- **NiteOS VPS** (`31.97.126.86`): Cloud Core — all Go microservices, Traefik (via nginx ingress), Postgres, Redis
- **Radio VPS** (`72.60.181.89`): Radio + Market + More — kept permanently separate; do not deploy NiteOS services here
- Each venue gets an **Edge Node** (Master Tablet or NiteBox) with pre-enrolled Android terminals

### VPS access model
- Non-root `niteops` user with sudo
- VS Code Remote SSH workflow
- GitHub is source of truth; VPS pulls from GitHub on deploy
- Pre-flight checklist before every service install: SSH keys, GitHub access, known_hosts, deploy key, firewall, Cloudflare API token

### Deployment path
- Phase 1: Manual or scripted pull-and-redeploy from VPS
- Phase 2: push-to-main triggers GitHub Actions → VPS deployment

### Pre-flight invariants (before any deployment)
1. SSH keys confirmed working
2. GitHub access confirmed, known_hosts pinned
3. Deploy key generated and installed (repo-scoped)
4. Firewall: only 22/80/443 public on NiteOS VPS
5. Cloudflare API token present (DNS-01 TLS)
6. TLS strategy confirmed and tested
