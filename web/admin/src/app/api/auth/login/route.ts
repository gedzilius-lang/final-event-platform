import { NextRequest, NextResponse } from 'next/server'
import { getSession } from '@/lib/session'
import { authLogin } from '@/lib/api'

export async function POST(req: NextRequest) {
  const { email, password } = await req.json()

  if (!email || !password) {
    return NextResponse.json({ error: 'Email and password required' }, { status: 400 })
  }

  try {
    const data = await authLogin(email, password)

    // Only allow venue_admin and nitecore roles into the admin console.
    // Decode the JWT payload to read the role claim (no verification needed —
    // the auth service already validated credentials; we trust its response).
    const [, payloadB64] = data.access_token.split('.')
    const payload = JSON.parse(Buffer.from(payloadB64, 'base64url').toString())
    const role: string = payload.role ?? ''

    if (role !== 'venue_admin' && role !== 'nitecore') {
      return NextResponse.json(
        { error: 'Insufficient permissions. venue_admin or nitecore role required.' },
        { status: 403 },
      )
    }

    const session = await getSession()
    session.userId = data.user_id
    session.role = role as 'venue_admin' | 'nitecore'
    session.venueId = payload.venue_id ?? undefined
    session.accessToken = data.access_token
    session.displayName = data.display_name || payload.display_name || email
    await session.save()

    return NextResponse.json({ ok: true })
  } catch (err: unknown) {
    const status = (err as { status?: number }).status ?? 500
    if (status === 401) {
      return NextResponse.json({ error: 'Invalid credentials' }, { status: 401 })
    }
    return NextResponse.json({ error: 'Login failed' }, { status: 500 })
  }
}
