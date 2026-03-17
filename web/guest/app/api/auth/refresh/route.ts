import { NextRequest, NextResponse } from 'next/server'
import { getSession } from '@/lib/session'
import { authRefresh, ApiError } from '@/lib/api'

// POST /api/auth/refresh — refreshes the access token using the stored refresh token.
// Called automatically when a 401 is encountered on server components.
export async function POST(req: NextRequest) {
  const session = await getSession()

  if (!session.refreshToken) {
    session.destroy()
    return NextResponse.json({ error: 'No refresh token' }, { status: 401 })
  }

  try {
    const data = await authRefresh(session.refreshToken)
    session.accessToken = data.access_token
    if (data.refresh_token) session.refreshToken = data.refresh_token
    await session.save()
    return NextResponse.json({ ok: true })
  } catch (err: unknown) {
    const e = err as ApiError
    // Refresh failed — clear session, user must log in again
    session.destroy()
    return NextResponse.json({ error: 'Session expired' }, { status: 401 })
  }
}
