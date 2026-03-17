// Next.js middleware — runs on the edge before every matched request.
// 1. Route protection: redirect unauthenticated users to /login.
// 2. CSRF: reject cross-origin POST/PUT/DELETE requests to /api/* routes.
import { NextRequest, NextResponse } from 'next/server'

export const config = {
  matcher: [
    // Protect these routes
    '/wallet/:path*',
    '/tickets/:path*',
    '/events/:path*',
    '/session/:path*',
    '/venues/:path*',
    // CSRF check on all BFF API routes
    '/api/:path*',
  ],
}

const PROTECTED_ROUTES = ['/wallet', '/tickets', '/events', '/session', '/venues']

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl
  const method = req.method

  // ── CSRF check for mutating BFF API routes ────────────────────────────────
  if (pathname.startsWith('/api/') && ['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
    // Health endpoint is exempt
    if (pathname === '/api/health') return NextResponse.next()
    // Stripe webhook is exempt (Stripe sends from different origin with signature)
    // — no webhook endpoint on guest BFF, but keep rule explicit

    const origin = req.headers.get('origin')
    const host = req.headers.get('host')
    if (origin) {
      const originHost = new URL(origin).host
      if (originHost !== host) {
        return NextResponse.json({ error: 'CSRF check failed' }, { status: 403 })
      }
    }
  }

  // ── Route protection for authenticated pages ──────────────────────────────
  const isProtected = PROTECTED_ROUTES.some((r) => pathname.startsWith(r))
  if (isProtected) {
    // iron-session cookie presence check (can't decrypt in edge middleware)
    const sessionCookie = req.cookies.get('niteos-guest-session')
    if (!sessionCookie?.value) {
      const loginUrl = new URL('/login', req.url)
      loginUrl.searchParams.set('next', pathname)
      return NextResponse.redirect(loginUrl)
    }
  }

  return NextResponse.next()
}
