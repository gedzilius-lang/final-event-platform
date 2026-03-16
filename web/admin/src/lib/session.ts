// Session management via iron-session (httpOnly cookies).
// The backend access token is stored server-side only — never exposed to client JS.
import { getIronSession, SessionOptions } from 'iron-session'
import { cookies } from 'next/headers'

export interface AdminSession {
  userId: string
  role: 'nitecore' | 'venue_admin'
  venueId?: string
  accessToken: string
  displayName: string
}

const sessionOptions: SessionOptions = {
  password: process.env.SESSION_SECRET!,
  cookieName: 'niteos-admin-session',
  cookieOptions: {
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'lax',
    maxAge: 60 * 60 * 12, // 12 hours
  },
}

export async function getSession() {
  return getIronSession<AdminSession>(await cookies(), sessionOptions)
}
