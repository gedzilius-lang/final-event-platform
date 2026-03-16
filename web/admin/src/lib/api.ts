// BFF API helper. All calls go server-side only — never from browser.
// Backend service URLs are injected from env vars (no NEXT_PUBLIC_ prefix).

const GATEWAY_URL = process.env.GATEWAY_URL ?? 'http://localhost:8000'

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

export async function backendFetch<T>(
  path: string,
  accessToken: string,
  init?: RequestInit,
): Promise<T> {
  const url = `${GATEWAY_URL}${path}`
  const res = await fetch(url, {
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

// ── Auth service (no token needed) ────────────────────────────────────────────

const AUTH_URL = process.env.AUTH_SERVICE_URL ?? 'http://localhost:8010'

export async function authLogin(email: string, password: string) {
  const res = await fetch(`${AUTH_URL}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
    cache: 'no-store',
  })
  if (!res.ok) {
    throw new ApiError(res.status, 'Invalid credentials')
  }
  return res.json() as Promise<{
    user_id: string
    access_token: string
    refresh_token: string
    expires_in: number
    display_name?: string
  }>
}

// ── Typed helpers ─────────────────────────────────────────────────────────────

export interface Venue {
  venue_id: string
  name: string
  slug: string
  city: string
  address?: string
  capacity: number
  timezone: string
  is_active: boolean
  created_at: string
}

export interface CatalogItem {
  item_id: string
  venue_id: string
  name: string
  category: string
  price_nc: number
  icon?: string
  stock_qty?: number
  is_active: boolean
  display_order: number
  happy_hour_price_nc?: number
}

export interface Device {
  device_id: string
  venue_id: string
  device_role: string
  device_name?: string
  status: string
  enrolled_at?: string
  last_heartbeat?: string
  last_seen_ip?: string
}

export interface VenueSession {
  session_id: string
  user_id: string
  venue_id: string
  nitetap_uid?: string
  opened_at: string
  closed_at?: string
  total_spend_nc: number
  status: string
}

export interface VenueRevenue {
  venue_id: string
  period_start: string
  period_end: string
  total_orders_nc: number
  total_topups_nc: number
  order_count: number
  session_count: number
}

export interface User {
  user_id: string
  email: string
  display_name?: string
  role: string
  venue_id?: string
  global_xp: number
  global_level: number
  created_at: string
}
