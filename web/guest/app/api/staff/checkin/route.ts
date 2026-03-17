// BFF: door staff check-in — opens a venue session for a guest.
import { NextRequest, NextResponse } from 'next/server'
import { requireSession } from '@/lib/session'
import { checkinGuest, ApiError } from '@/lib/api'

const ALLOWED_ROLES = ['door_staff', 'venue_admin', 'nitecore']

export async function POST(req: NextRequest) {
  let session
  try {
    session = await requireSession()
  } catch {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
  }

  if (!ALLOWED_ROLES.includes(session.role)) {
    return NextResponse.json({ error: 'Forbidden' }, { status: 403 })
  }

  const body = await req.json().catch(() => null)
  if (!body?.user_id || !body?.venue_id) {
    return NextResponse.json({ error: 'user_id and venue_id required' }, { status: 400 })
  }

  try {
    const venueSession = await checkinGuest(
      body.user_id,
      body.venue_id,
      body.nfc_uid ?? '',
      session.accessToken,
    )
    return NextResponse.json(venueSession, { status: 201 })
  } catch (err) {
    if (err instanceof ApiError) {
      return NextResponse.json({ error: err.message }, { status: err.status })
    }
    return NextResponse.json({ error: 'Internal error' }, { status: 500 })
  }
}
