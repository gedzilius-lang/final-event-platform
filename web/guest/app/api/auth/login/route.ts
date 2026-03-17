import { NextRequest, NextResponse } from 'next/server'
import { getSession } from '@/lib/session'
import { authLogin, ApiError } from '@/lib/api'

export async function POST(req: NextRequest) {
  const { email, password } = await req.json()

  if (!email || !password) {
    return NextResponse.json({ error: 'Email and password required' }, { status: 400 })
  }

  try {
    const data = await authLogin(email, password)

    // Decode role from JWT payload (no verification — auth service already did that).
    const [, payloadB64] = data.access_token.split('.')
    const payload = JSON.parse(Buffer.from(payloadB64, 'base64url').toString())

    const session = await getSession()
    session.userId = data.user_id
    session.accessToken = data.access_token
    session.refreshToken = data.refresh_token ?? ''
    session.displayName = data.display_name || payload.display_name || email
    session.role = payload.role ?? 'guest'
    session.venueId = payload.venue_id ?? undefined
    await session.save()

    return NextResponse.json({ ok: true })
  } catch (err: unknown) {
    const status = (err as ApiError).status ?? 500
    if (status === 401) {
      return NextResponse.json({ error: 'Invalid credentials' }, { status: 401 })
    }
    return NextResponse.json({ error: 'Login failed' }, { status: 500 })
  }
}
