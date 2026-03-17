// BFF: look up a guest by email — door/security staff only.
import { NextRequest, NextResponse } from 'next/server'
import { requireSession } from '@/lib/session'
import { getUserByEmail, ApiError } from '@/lib/api'

const ALLOWED_ROLES = ['door_staff', 'venue_admin', 'security', 'nitecore']

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

  const email = req.nextUrl.searchParams.get('email')
  if (!email) {
    return NextResponse.json({ error: 'email required' }, { status: 400 })
  }

  try {
    const profile = await getUserByEmail(email, session.accessToken)
    return NextResponse.json(profile)
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.status === 404) return NextResponse.json({ error: 'Guest not found' }, { status: 404 })
      return NextResponse.json({ error: 'Lookup failed' }, { status: err.status })
    }
    return NextResponse.json({ error: 'Internal error' }, { status: 500 })
  }
}
