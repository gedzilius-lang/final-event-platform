# BUSINESS_PLAN.md

The business model, revenue mechanics, market positioning, regulatory strategy, and go-to-market plan for NiteOS. This is a working document for the founding team — not a pitch deck.

---

## The Business in One Sentence

NiteOS captures the transaction layer of nightlife venues by replacing fragmented POS, ticketing, guestlist, and loyalty tools with a single offline-capable operating system that pre-captures guest spend, speeds up service, and gives operators data they currently cannot get from any single tool.

---

## The Market Problem

Swiss nightlife venues currently run on:
- A generic POS (SumUp, Trivec, Toast, or a till)
- A separate ticketing tool (Eventfrog, TicketingHub, Stager, or manual guestlists)
- A separate guestlist and reservation system
- Spreadsheets for inventory
- WhatsApp for staff coordination
- Nothing for guest loyalty beyond a stamp card or email list

Every one of these systems is siloed. None of them talk to each other. A venue operator has no single view of what happened on Saturday night, who their best customers are, which bartender is slowest, or how much pre-committed spend is waiting to be poured.

The structural pain is **queue time and throughput**. Bar revenue is a function of pours per hour. Card terminals add 2–4 seconds per transaction in authorization latency. At a bar doing 200 transactions per hour, that is 6-13 minutes of dead time per hour at that station alone. At the bar level, faster payment infrastructure is directly monetizable — it is not a feature, it is revenue.

TWINT has over 6 million Swiss users and >773 million transactions in 2024. The Swiss population is digitally payment-ready. The missing piece is not willingness — it is a system designed for bar-speed, venue-scale, offline-capable operation.

---

## Why This Wins

**Speed:** Tap-to-pay at NiteKiosk is faster than any card terminal. The guest's NiteTap is already loaded. There is no card insertion, no PIN, no authorization wait. The average POS interaction drops from ~8 seconds to ~2 seconds.

**Pre-committed spend (The Float):** When a guest tops up NiteCoins, the cash is captured before a single drink is poured. The venue's working capital position is fundamentally different from traditional cash-on-delivery service. Venues see this money before they need it.

**Offline reliability:** Every competitor relies on cloud connectivity. NiteOS's edge node means the venue keeps operating when Wi-Fi fails. This is the product's hardest-to-copy differentiator. Building it requires significant engineering investment; copying it requires understanding why it matters — which most POS companies have never had reason to do.

**Data moat:** Every transaction, check-in, order, and loyalty interaction flows through NiteOS. Over time, this creates an operational dataset that the venue cannot get anywhere else — and that Nitecore can eventually use for cross-venue benchmarking and predictive recommendations.

**Switching cost:** Once a venue runs nightly operations through NiteOS — catalog configured, staff trained, hardware installed, guest base on NiteTap — switching away is genuinely painful. It is not like cancelling a SaaS subscription. It is like ripping out the electrical system.

---

## Revenue Streams

### 1. SaaS Platform Fee
Monthly recurring license for the NiteOS platform.

| Tier | Monthly Fee | Included |
|------|------------|---------|
| Core | ~500 CHF/mo | 1 venue, up to 3 kiosk devices, basic reporting |
| Growth | ~1,200 CHF/mo | 1 venue, up to 8 devices, full reporting, CRM tools |
| Multi-Venue | Custom | Multiple venues, network analytics, dedicated support |

These numbers are directional. Actual pricing depends on pilot data and competitive pressure. The principle: price to the **operational value delivered** (faster service = more revenue per night), not to software vanity.

### 2. Transaction Fee on Top-Ups
1.5% – 2.5% on every fiat-to-NC conversion (TWINT or Stripe).

A venue processing 10,000 CHF per night in wallet top-ups generates 150–250 CHF per night in transaction fees. At 20 operating nights per month, that is 3,000–5,000 CHF/month per venue in transaction revenue — on top of the SaaS fee.

At network scale (10 venues × 10,000 CHF/night average), this is 30,000–50,000 CHF/month in pure transaction revenue.

The fee is taken at the moment of top-up via Stripe Connect:
- 95 CHF goes to Venue's connected account
- 5 CHF (5% example) goes to Nitecore

The rate is negotiated per venue in the partnership contract. High-volume venues get lower rates as part of an enterprise deal.

### 3. Breakage Revenue
NiteCoins that expire after 365 days of inactivity generate breakage revenue.

Breakage is split contractually between Nitecore and the Venue Partner. A typical split: 50/50. The venue is made aware of this revenue stream during onboarding — it is positioned as a benefit, not a hidden charge. Venues have incentive to issue bonus NC (driving top-ups) because they share in the upside when some of those credits expire.

Breakage rates at entertainment venues typically range from 5% to 15% of outstanding balance. At a 10% breakage rate on 50,000 CHF monthly float across 10 venues, Nitecore's share (~50%) is 2,500 CHF/month — growing as the network scales.

**Regulatory note:** Under Swiss law, the liability for expired voucher proceeds rests with the party that issued the voucher. The Venue Partner Agreement explicitly assigns tax and regulatory liability for breakage to the Venue, not to Nitecore. This is a standard MPV structure.

### 4. Hardware-as-a-Service (HaaS)
Venue hardware is never sold outright. It is leased.

Partnership model: Nitecore sells hardware units to a leasing partner (Grenke or equivalent). Grenke leases to the venue.

**Approximate monthly hardware fee: ~300 CHF/mo per venue** for a standard kit:
- 1× Master Tablet (Samsung Galaxy Tab Active 5 or equivalent)
- 2× NiteKiosk phones
- 1× NiteTerminal phone
- Initial NiteTap card stock (50–100 units)
- UniFi network switch or access point

Why leasing beats selling:
- Removes the "10,000 CHF upfront" CapEx objection
- Creates predictable monthly revenue
- Increases switching cost (venue must return hardware to stop)
- Aligns Nitecore incentives with ongoing reliability (broken hardware = Nitecore's problem)

### 5. Professional Services
One-time revenue from venue onboarding, initial setup, catalog configuration, staff training, and deployment support.

- Standard onboarding: 1,500–3,000 CHF one-time
- Enterprise/complex venue: custom

This is not the moat. It funds early cash flow and builds operational playbooks for later.

---

## Unit Economics (Per Venue)

**Year 1 target venue (pilot-grade):**
- SaaS: 500 CHF/mo
- Hardware: 300 CHF/mo
- Transactions: assume 5,000 CHF/mo in top-ups × 2% = 100 CHF/mo
- **Minimum recurring per venue: ~900 CHF/mo**

**Year 2 target venue (growth-grade):**
- SaaS: 1,200 CHF/mo
- Hardware: 300 CHF/mo
- Transactions: assume 20,000 CHF/mo in top-ups × 2% = 400 CHF/mo
- Breakage share: ~150 CHF/mo (conservative)
- **Target recurring per venue: ~2,050 CHF/mo**

**10 venues, Year 2:** ~20,500 CHF/month recurring revenue. This covers engineering + ops costs with margin at a lean team size.

**Why 10 venues matters:** The network effect begins here. Cross-venue data becomes meaningful. Benchmarking becomes a product feature. Partner negotiation leverage with Stripe/TWINT improves.

---

## Regulatory Strategy

### Phase 1 (2026, pilot): The Sandbox

- **Legal structure of NiteCoin:** Multi-Purpose Voucher (MPV). Not e-money. Not a deposit.
- **Swiss FinMA Sandbox:** Operate under the "Limited Network Exception." Total public deposits (outstanding NiteCoin float) must stay under **1,000,000 CHF** (the sandbox cap).
- **Single venue pilot** keeps float well within 100,000 CHF.
- No automated settlement in Phase 1. Venue keeps all cash directly. Nitecore invoices the venue manually for SaaS + transaction fee.
- **Minimum compliance actions in Phase 1:**
  - Document the voucher structure in the Venue Partner Agreement
  - Establish clear terms: no cash-out, limited network, 365-day expiry
  - Basic KYC/AML for registered accounts above a transaction threshold (GwG compliance baseline)

### Phase 2 (Q3/Q4 2026, 3–5 venues): Activate Settlement

- **SRO Membership:** Join VQF (Financial Services Standards Association) to comply formally with AML obligations (GwG) as float grows
- **Stripe Connect:** Activate automated fund splitting. Each top-up triggers automatic routing — venue's share to venue account, Nitecore's share to Nitecore account
- **Data protection:** nDSG (New Swiss Federal Act on Data Protection) compliance. Physical NiteTap bands are anonymous. Names collected only for registered accounts.

### Phase 3 (deferred): Scale

- Triggered when float exceeds 1,000,000 CHF across the network
- Requires formal engagement with FinMA
- Possible outcome: simplified e-money license or formal MPV issuer registration
- Until this threshold is reached, Phase 2 structure is sufficient

### Hard legal rules that never change regardless of phase:
1. NiteCoin cannot be cashed out by guests
2. NiteCoin cannot be transferred to accounts outside the NiteOS network
3. No NC is minted without a verified payment callback
4. Float cap monitoring must be automated and reported to operations monthly

---

## Customer Profile

### Primary buyer (the decision-maker)
- Nightclub owner or general manager
- Venue capacity: 200–1,500 guests
- Recurring events (not one-off): same venue, weekly or bi-weekly nights
- Current pain: bar queues, fragmented tools, no data, cash handling risks
- Orientation: willing to invest in operational infrastructure; does not need to be "tech forward"

### Wrong early customer
- Tiny bars with <50 covers and no queue problem
- Venues that want broad customization before any proof
- Operators running cash-only by conviction or legal avoidance
- One-off event organizers (festivals, weddings, corporate) — different product category

### Secondary buyers / influencers
- Venue promoters (benefit from ticket pre-sales and guest data)
- Bar managers (benefit from faster service and automated inventory)
- Hospitality groups (benefit from multi-venue analytics)

### End users (guests)
- Club-going demographic in Zurich, 20–35
- Already uses TWINT and digital payments
- Responds to loyalty incentives and status signals
- Does not want friction at entry

---

## Go-To-Market

### Phase 1 — Prove It (Q1/Q2 2026)

One pilot venue. Ideally "Supermarket" Zurich (already in the codebase as a seeded venue) or equivalent.

**Goal:** Not scale. Not revenue. Proof.

What must be proven:
- Venue runs through a full Saturday night without system failure
- Bar throughput increases measurably (even +10% is signal)
- Staff uses devices without significant confusion
- Offline fallback works when tested deliberately
- Check-in flow is faster than existing door process
- Venue admin can see useful data the next morning

**Success = one venue that would not give NiteOS up.**

Pricing in Phase 1: heavily discounted or free for the pilot venue. The cost is engineering time. The return is a real-world reference case.

### Phase 2 — Replicate It (Q3/Q4 2026)

Target: 3–5 venues in Zurich (same city, same legal/tax regime).

**Goal:** A repeatable deployment model and a support playbook.

What must be built:
- Onboarding playbook (catalog configuration, device enrollment, staff training)
- Deployment kit (hardware unboxing to operational in under 4 hours)
- Support process (what to do at 1 AM when something breaks)
- Pricing model confirmed by market reaction

Activate Stripe Connect and SRO/VQF membership in this phase.

### Phase 3 — Expand (2027+)

Only after Phase 2 demonstrates:
- Reliable deployment (not just once, but consistently)
- Clear support cost per venue
- Verified revenue model

Expansion to other Swiss cities (Basel, Geneva, Lausanne), then possibly Liechtenstein and Austria (similar legal/cultural context), then broader German-speaking market.

---

## Competitive Positioning

### NiteOS is NOT competing with:
- **SumUp / Square / Trivec** on POS fees — their customers want a terminal. NiteOS customers want a system.
- **Tappit / Glownet / Billfold** on wristband cashless — they target festivals. NiteOS is for permanent venues.
- **TicketFairy / Stager** on ticketing — they control the ticketing. NiteOS owns the identity and the spend after the ticket is bought.

### NiteOS IS competing with:
- The venue's inertia and preference for "what we've always done"
- TWINT QR codes taped to the bar (the "good enough" baseline)
- The Access Group nightclub suite (the only direct software competitor with a similar scope)

### The positioning statement:
"NiteOS is the operating system for your venue. It is not a payment tool. It is not a loyalty app. It is the system your staff uses to check people in, sell drinks, and run the night — and it keeps working when the internet doesn't."

---

## The People We Like Brand

People We Like is not a product — it is a cultural identity. It gives NiteOS a face.

The brand roles:
- **People We Like Radio**: ambient presence, cultural credibility, scene identity. Listeners are the target demographic. The radio builds audience before the product launches.
- **People We Like Market**: commercial brand expression. Curated goods, editorial drops, taste graph. Generates revenue independently and establishes the brand's aesthetic authority.
- **NiteOS under the People We Like umbrella**: venues deploying NiteOS are "People We Like partners." The platform creates a sense of being part of a curated network, not just buying software.

Brand separation rule: People We Like Radio and Market have their own P&L. They are not subsidized by NiteOS. NiteOS is not subsidized by them. They share the brand and potentially the guest identity — nothing else.

---

## Financial Summary

| Phase | Target | Monthly Recurring | Key Milestone |
|-------|--------|------------------|---------------|
| Phase 1 (Pilot) | 1 venue | ~0–500 CHF | Survive a Saturday night |
| Phase 2 (Network) | 3–5 venues | ~3,000–10,000 CHF | Repeatable deployment + Stripe Connect live |
| Phase 3 (Scale) | 10+ venues | ~20,000+ CHF | NiteOS is the default operating stack for Swiss nightlife |

Breakeven at lean team size (2–3 engineers): approximately 15,000 CHF/month recurring. Achievable at 8–10 venues with a mix of SaaS, transaction fees, and hardware.
