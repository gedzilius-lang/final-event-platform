// CSRF protection: reject mutating requests from unexpected origins.
// Applied to all API routes. Pattern from MIGRATION_PLAN.md M4.2.
import { NextRequest, NextResponse } from 'next/server'

const ALLOWED_ORIGIN = process.env.ADMIN_ORIGIN ?? 'http://localhost:3001'

function isMutating(method: string) {
  return ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method.toUpperCase())
}

export function middleware(req: NextRequest) {
  if (req.nextUrl.pathname.startsWith('/api/') && isMutating(req.method)) {
    const origin = req.headers.get('origin') ?? ''
    if (origin !== ALLOWED_ORIGIN) {
      return NextResponse.json({ error: 'csrf' }, { status: 403 })
    }
  }
  return NextResponse.next()
}

export const config = {
  matcher: '/api/:path*',
}
