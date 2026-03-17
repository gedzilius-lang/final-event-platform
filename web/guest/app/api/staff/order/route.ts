// BFF: create a pending order — bartender/venue_admin/nitecore only.
import { NextRequest, NextResponse } from 'next/server'
import { requireSession } from '@/lib/session'
import { createOrder, ApiError } from '@/lib/api'

const ALLOWED_ROLES = ['bartender', 'venue_admin', 'nitecore']

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
  if (!body?.venue_id || !body?.items?.length || !body?.idempotency_key) {
    return NextResponse.json({ error: 'venue_id, items, and idempotency_key required' }, { status: 400 })
  }

  try {
    const order = await createOrder(
      {
        venue_id: body.venue_id,
        guest_session_id: body.guest_session_id,
        items: body.items,
        idempotency_key: body.idempotency_key,
      },
      session.accessToken,
    )
    return NextResponse.json(order, { status: 201 })
  } catch (err) {
    if (err instanceof ApiError) {
      return NextResponse.json({ error: err.message }, { status: err.status })
    }
    return NextResponse.json({ error: 'Internal error' }, { status: 500 })
  }
}
