// BFF: get active session for a user — security/door/manager only.
import { NextRequest, NextResponse } from 'next/server'
import { requireSession } from '@/lib/session'
import { getActiveSession, ApiError } from '@/lib/api'

const ALLOWED_ROLES = ['door_staff', 'security', 'venue_admin', 'nitecore']

export async function GET(req: NextRequest) {
  let session
  try {
    session = await requireSession()
  } catch {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  if (!ALLOWED_ROLES.includes(session.role)) {
    return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
  }

  const userId = req.nextUrl.searchParams.get('user_id')
  if (!userId) {
    return NextResponse.json({ error: 'user_id required' }, { status: 400 })
  }

  try {
    const activeSession = await getActiveSession(userId, session.accessToken)
    if (!activeSession) return NextResponse.json(null, { status: 404 })
    return NextResponse.json(activeSession)
  } catch (err) {
    if (err instanceof ApiError) {
      return NextResponse.json({ error: err.message }, { status: err.status })
    }
    return NextResponse.json({ error: 'Internal error' }, { status: 500 })
  }
}
