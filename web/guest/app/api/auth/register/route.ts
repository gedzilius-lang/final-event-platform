import { NextRequest, NextResponse } from 'next/server'
import { getSession } from '@/lib/session'
import { authRegister, ApiError } from '@/lib/api'

export async function POST(req: NextRequest) {
  const { email, password, display_name } = await req.json()

  if (!email || !password || !display_name) {
    return NextResponse.json(
      { error: 'Email, password and display_name required' },
      { status: 400 },
    )
  }

  try {
    const data = await authRegister(email, password, display_name)

    const [, payloadB64] = data.access_token.split('.')
    const payload = JSON.parse(Buffer.from(payloadB64, 'base64url').toString())

    const session = await getSession()
    session.userId = data.user_id
    session.accessToken = data.access_token
    session.refreshToken = data.refresh_token ?? ''
    session.displayName = data.display_name || payload.display_name || display_name
    session.role = payload.role ?? 'guest'
    await session.save()

    return NextResponse.json({ ok: true })
  } catch (err: unknown) {
    const e = err as ApiError
    if (e.status === 409) {
      return NextResponse.json({ error: 'Email already registered' }, { status: 409 })
    }
    return NextResponse.json({ error: e.message || 'Registration failed' }, { status: e.status ?? 500 })
  }
}
