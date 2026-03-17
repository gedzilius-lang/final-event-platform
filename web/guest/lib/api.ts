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

// 1 CHF = 100 NC (backend domain: CHFToNC(chf) = int(chf * 100))
export function ncToCHF(nc: number): string {
  return (nc / 100).toFixed(2)
}

export function chfToNC(chf: number): number {
  return Math.round(chf * 100)
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
