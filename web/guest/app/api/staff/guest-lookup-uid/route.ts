// BFF: look up a guest by NiteTap UID — bar/door/security staff.
// Calls GET /profiles/users/by-nfc-uid/{uid} via gateway.
import { NextRequest, NextResponse } from 'next/server'
import { requireSession } from '@/lib/session'
import { backendFetch, UserProfile, ApiError } from '@/lib/api'

const ALLOWED_ROLES = ['bartender', 'door_staff', 'security', 'venue_admin', 'nitecore']

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

  const uid = req.nextUrl.searchParams.get('uid')
  if (!uid) {
    return NextResponse.json({ error: 'uid required' }, { status: 400 })
  }

  try {
    const profile = await backendFetch<UserProfile>(
      `/profiles/users/by-nfc-uid/${encodeURIComponent(uid)}`,
      session.accessToken,
    )
    return NextResponse.json(profile)
  } catch (err) {
    if (err instanceof ApiError) {
      return NextResponse.json({ error: err.message }, { status: err.status })
    }
    return NextResponse.json({ error: 'internal error' }, { status: 500 })
  }
}
