import { NextRequest, NextResponse } from 'next/server'
import { getSession } from '@/lib/session'
import { backendFetch } from '@/lib/api'

export async function GET(
  _req: NextRequest,
  { params }: { params: { venue_id: string; item_id: string } },
) {
  const session = await getSession()
  if (!session.userId) return NextResponse.json({ error: 'unauthorized' }, { status: 401 })
  try {
    const data = await backendFetch(
      `/catalog/venues/${params.venue_id}/items/${params.item_id}`,
      session.accessToken,
    )
    return NextResponse.json(data)
  } catch (e: unknown) {
    const err = e as { status?: number; message?: string }
    return NextResponse.json({ error: err.message }, { status: err.status ?? 500 })
  }
}

export async function DELETE(
  _req: NextRequest,
  { params }: { params: { venue_id: string; item_id: string } },
) {
  const session = await getSession()
  if (!session.userId) return NextResponse.json({ error: 'unauthorized' }, { status: 401 })
  try {
    await backendFetch(
      `/catalog/venues/${params.venue_id}/items/${params.item_id}`,
      session.accessToken,
      { method: 'DELETE' },
    )
    return NextResponse.json({ status: 'deleted' })
  } catch (e: unknown) {
    const err = e as { status?: number; message?: string }
    return NextResponse.json({ error: err.message }, { status: err.status ?? 500 })
  }
}
