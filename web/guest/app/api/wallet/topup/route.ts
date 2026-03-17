import { NextRequest, NextResponse } from 'next/server'
import { requireSession } from '@/lib/session'
import { createTopupIntent, ApiError } from '@/lib/api'
import { randomUUID } from 'crypto'

export async function POST(req: NextRequest) {
  let session
  try {
    session = await requireSession()
  } catch {
    return NextResponse.json({ error: 'Unauthenticated' }, { status: 401 })
  }

  const body = await req.json()
  const amount_chf = Number(body.amount_chf)

  if (!amount_chf || amount_chf < 5 || amount_chf > 500) {
    return NextResponse.json(
      { error: 'amount_chf must be between 5 and 500' },
      { status: 400 },
    )
  }

  // Generate idempotency key server-side — prevents duplicate intents on retries
  const idempotencyKey = `guest:${session.userId}:${randomUUID()}`

  try {
    const intent = await createTopupIntent(
      session.userId,
      amount_chf,
      idempotencyKey,
      session.accessToken,
    )
    return NextResponse.json(intent)
  } catch (err: unknown) {
    const e = err as ApiError
    return NextResponse.json(
      { error: e.message || 'Topup failed' },
      { status: e.status ?? 500 },
    )
  }
}
