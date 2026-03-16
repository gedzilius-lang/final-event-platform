import { NextRequest, NextResponse } from 'next/server'
import { getSession } from '@/lib/session'
import { backendFetch } from '@/lib/api'

export async function PATCH(
  req: NextRequest,
  { params }: { params: { user_id: string } },
) {
  const session = await getSession()
  if (!session.userId) return NextResponse.json({ error: 'unauthorized' }, { status: 401 })
  if (session.role !== 'nitecore') {
    return NextResponse.json({ error: 'nitecore role required' }, { status: 403 })
  }
  const body = await req.json()
  try {
    const data = await backendFetch(
      `/profiles/users/${params.user_id}/venue`,
      session.accessToken,
      { method: 'PATCH', body: JSON.stringify(body) },
    )
    return NextResponse.json(data)
  } catch (e: unknown) {
    const err = e as { status?: number; message?: string }
    return NextResponse.json({ error: err.message }, { status: err.status ?? 500 })
  }
}
