// BFF: look up a guest by NiteTap UID — bar/door/security staff.
// GAP: profiles service does not yet expose GET /users/by-nitetap/{uid}.
// This endpoint stubs the lookup by returning a minimal "anonymous" response.
// When the profiles service adds UID lookup, update this route to call it.
import { NextRequest, NextResponse } from 'next/server'
import { requireSession } from '@/lib/session'

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

  // STUB: profiles service does not yet expose a by-NFC-UID lookup endpoint.
  // Return 404 so callers fall back to anonymous order.
  // TODO: call GET /profiles/users/by-nfc-uid/{uid} when available.
  return NextResponse.json(
    { error: 'NiteTap UID lookup not yet available — order will be anonymous' },
    { status: 404 },
  )
}
