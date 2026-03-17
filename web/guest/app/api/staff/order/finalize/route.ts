// BFF: finalize (charge) an order — bartender/venue_admin/nitecore only.
import { NextRequest, NextResponse } from 'next/server'
import { requireSession } from '@/lib/session'
import { finalizeOrder, ApiError } from '@/lib/api'

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
  if (!body?.order_id) {
    return NextResponse.json({ error: 'order_id required' }, { status: 400 })
  }

  try {
    const order = await finalizeOrder(body.order_id, body.guest_user_id ?? '', session.accessToken)
    return NextResponse.json(order)
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.status === 402) {
        return NextResponse.json({ error: 'Insufficient balance' }, { status: 402 })
      }
      return NextResponse.json({ error: err.message }, { status: err.status })
    }
    return NextResponse.json({ error: 'Internal error' }, { status: 500 })
  }
}
