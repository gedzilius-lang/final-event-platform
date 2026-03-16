import { NextRequest, NextResponse } from 'next/server'
import { getSession } from '@/lib/session'
import { backendFetch } from '@/lib/api'

export async function GET() {
  const session = await getSession()
  if (!session.userId) return NextResponse.json({ error: 'unauthorized' }, { status: 401 })
  try {
    const data = await backendFetch('/catalog/venues', session.accessToken)
    return NextResponse.json(data)
  } catch (e: unknown) {
    const err = e as { status?: number; message?: string }
    return NextResponse.json({ error: err.message }, { status: err.status ?? 500 })
  }
}

export async function POST(req: NextRequest) {
  const session = await getSession()
  if (!session.userId) return NextResponse.json({ error: 'unauthorized' }, { status: 401 })
  if (session.role !== 'nitecore') {
    return NextResponse.json({ error: 'nitecore role required' }, { status: 403 })
  }
  const body = await req.json()
  try {
    const data = await backendFetch('/catalog/venues', session.accessToken, {
      method: 'POST',
      body: JSON.stringify(body),
    })
    return NextResponse.json(data, { status: 201 })
  } catch (e: unknown) {
    const err = e as { status?: number; message?: string }
    return NextResponse.json({ error: err.message }, { status: err.status ?? 500 })
  }
}
