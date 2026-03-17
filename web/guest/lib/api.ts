// BFF API helpers — server-side only. Never imported from client components.

const GATEWAY_URL = process.env.GATEWAY_URL ?? 'http://localhost:8000'
const AUTH_SERVICE_URL = process.env.AUTH_SERVICE_URL ?? 'http://localhost:8010'

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
  }
}

export async function backendFetch<T>(
  path: string,
  accessToken: string,
  init?: RequestInit,
): Promise<T> {
  const res = await fetch(`${GATEWAY_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${accessToken}`,
      ...(init?.headers ?? {}),
    },
    cache: 'no-store',
  })
  if (!res.ok) {
    const body = await res.text()
    throw new ApiError(res.status, body)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

// ── Auth (direct to auth service — no JWT required) ───────────────────────────

export async function authLogin(email: string, password: string) {
  const res = await fetch(`${AUTH_SERVICE_URL}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
    cache: 'no-store',
  })
  if (!res.ok) throw new ApiError(res.status, 'Invalid credentials')
  return res.json() as Promise<AuthResponse>
}

export async function authRegister(
  email: string,
  password: string,
  displayName: string,
) {
  const res = await fetch(`${AUTH_SERVICE_URL}/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password, display_name: displayName }),
    cache: 'no-store',
  })
  if (!res.ok) {
    const body = await res.text()
    throw new ApiError(res.status, body)
  }
  return res.json() as Promise<AuthResponse>
}

export async function authRefresh(refreshToken: string) {
  const res = await fetch(`${AUTH_SERVICE_URL}/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
    cache: 'no-store',
  })
  if (!res.ok) throw new ApiError(res.status, 'Refresh failed')
  return res.json() as Promise<AuthResponse>
}

// ── NC / CHF conversion ───────────────────────────────────────────────────────

// 1 NC = 1 CHF — fixed peg (PRODUCT_BLUEPRINT §NiteCoin).
export function ncToCHF(nc: number): string {
  return nc.toFixed(2)
}

export function chfToNC(chf: number): number {
  return Math.round(chf)
}

// ── Types ─────────────────────────────────────────────────────────────────────

export interface AuthResponse {
  user_id: string
  access_token: string
  refresh_token?: string
  expires_in?: number
  display_name?: string
}

export interface WalletBalance {
  user_id: string
  balance_nc: number
}

export interface Ticket {
  ticket_id: string
  user_id: string
  venue_id: string
  event_name: string
  valid_from: string
  valid_until: string
  status: string // 'active' | 'used' | 'expired' | 'cancelled'
  qr_payload: string
  created_at: string
}

export interface VenueSummary {
  venue_id: string
  name: string
  slug: string
  city: string
  is_active: boolean
  upcoming_events?: EventSummary[]
}

export interface EventSummary {
  event_id?: string
  venue_id: string
  event_name: string
  date: string
  capacity?: number
  ticket_price_nc?: number
}

export interface VenueSession {
  session_id: string
  user_id: string
  venue_id: string
  nitetap_uid?: string
  opened_at: string
  closed_at?: string
  total_spend_nc: number
  status: string // 'open' | 'closed'
}

export interface TopupIntent {
  topup_id: string
  client_secret?: string  // Stripe: use with Stripe.js
  redirect_url?: string   // TWINT: redirect user here
  provider: string
  amount_chf: number
  amount_nc: number
}

export interface UserProfile {
  user_id: string
  display_name: string
  email?: string
  role: string
  venue_id?: string
  global_xp: number
  global_level: number
  created_at?: string
}

export interface LedgerEvent {
  event_id: string
  event_type: string // 'topup_confirmed' | 'order_paid' | 'venue_checkin' | 'session_closed' etc.
  amount_nc: number
  venue_id?: string
  reference_id?: string
  occurred_at: string
}

export interface Order {
  order_id: string
  venue_id: string
  staff_user_id: string
  guest_session_id?: string
  total_nc: number
  status: string // 'pending' | 'paid' | 'voided'
  created_at: string
}

export interface OrderItem {
  item_id: string
  name: string
  quantity: number
  price_nc: number
}

export interface CreateOrderRequest {
  venue_id: string
  guest_session_id?: string
  items: OrderItem[]
  idempotency_key: string
}

// ── API helpers ───────────────────────────────────────────────────────────────

export async function getWalletBalance(userId: string, token: string): Promise<WalletBalance> {
  return backendFetch<WalletBalance>(`/wallet/${userId}`, token)
}

export async function getMyTickets(userId: string, token: string): Promise<{ tickets: Ticket[] }> {
  return backendFetch<{ tickets: Ticket[] }>(`/ticketing/users/${userId}/tickets`, token)
}

export async function getVenues(token: string): Promise<{ venues: VenueSummary[] }> {
  return backendFetch<{ venues: VenueSummary[] }>('/catalog/venues', token)
}

export async function getActiveSession(userId: string, token: string): Promise<VenueSession | null> {
  try {
    return await backendFetch<VenueSession>(`/sessions/guest/${userId}`, token)
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null
    throw err
  }
}

export async function getUserProfile(userId: string, token: string): Promise<UserProfile> {
  return backendFetch<UserProfile>(`/profiles/users/${userId}`, token)
}

// Wallet transaction history — may return 404 if not yet implemented.
export async function getWalletHistory(userId: string, token: string): Promise<{ events: LedgerEvent[] }> {
  return backendFetch<{ events: LedgerEvent[] }>(`/wallet/${userId}/history`, token)
}

// Active sessions for a venue — door/manager staff only.
export async function listActiveSessions(venueId: string, token: string): Promise<{ sessions: VenueSession[]; count: number }> {
  return backendFetch<{ sessions: VenueSession[]; count: number }>(`/sessions/venues/${venueId}/active`, token)
}

// Look up a user by email — door/security staff only.
export async function getUserByEmail(email: string, token: string): Promise<UserProfile> {
  return backendFetch<UserProfile>(`/profiles/users/by-email/${encodeURIComponent(email)}`, token)
}

// Check in a guest — door staff only.
export async function checkinGuest(
  userId: string,
  venueId: string,
  nfcUid: string,
  token: string,
): Promise<VenueSession> {
  return backendFetch<VenueSession>('/sessions/checkin', token, {
    method: 'POST',
    body: JSON.stringify({ user_id: userId, venue_id: venueId, nfc_uid: nfcUid }),
  })
}

// Close (checkout) a session — door staff or guest.
export async function checkoutSession(sessionId: string, token: string): Promise<VenueSession> {
  return backendFetch<VenueSession>(`/sessions/${sessionId}/checkout`, token, { method: 'POST' })
}

// Create a pending order — bartender only. Returns order_id for finalize.
export async function createOrder(
  req: CreateOrderRequest,
  token: string,
): Promise<Order> {
  return backendFetch<Order>('/orders/', token, {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// Finalize (charge) an order — bartender only. Deducts NC from guest wallet.
export async function finalizeOrder(orderId: string, guestUserId: string, token: string): Promise<Order> {
  return backendFetch<Order>(`/orders/${orderId}/finalize`, token, {
    method: 'POST',
    body: JSON.stringify({ guest_user_id: guestUserId }),
  })
}

export async function createTopupIntent(
  userId: string,
  amountChf: number,
  idempotencyKey: string,
  token: string,
): Promise<TopupIntent> {
  return backendFetch<TopupIntent>(
    `/payments/topup/intent`,
    token,
    {
      method: 'POST',
      body: JSON.stringify({
        user_id: userId,
        amount_chf: amountChf,
        idempotency_key: idempotencyKey,
      }),
    },
  )
}
