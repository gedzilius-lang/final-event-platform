// Guest session — stored server-side in httpOnly cookie.
// The access token never touches client JavaScript.
import { getIronSession, SessionOptions } from 'iron-session'
import { cookies } from 'next/headers'

export interface GuestSession {
  userId: string
  accessToken: string
  refreshToken: string
  displayName: string
  role: string // 'guest' | 'venue_admin' | etc.
  venueId?: string
  // Set when the guest is in an active venue session (post NiteTap check-in).
  activeSessionId?: string
}

const sessionOptions: SessionOptions = {
  password: process.env.SESSION_SECRET!,
  cookieName: 'niteos-guest-session',
  cookieOptions: {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'lax',
    maxAge: 60 * 60 * 24, // 24 hours
  },
}

export async function getSession() {
  return getIronSession<GuestSession>(await cookies(), sessionOptions)
}

export async function requireSession(): Promise<GuestSession> {
  const session = await getSession()
  if (!session.userId || !session.accessToken) {
    throw new Error('unauthenticated')
  }
  return session
}
