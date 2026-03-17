// Profile page — identity, level, NiteTap UID, activity history.
import { redirect } from 'next/navigation'
import Link from 'next/link'
import { getSession } from '@/lib/session'
import {
  getUserProfile,
  getWalletBalance,
  getWalletHistory,
  getActiveSession,
  ApiError,
} from '@/lib/api'
import NavBar from '@/components/NavBar'

const XP_PER_LEVEL = 500

export default async function ProfilePage() {
  const session = await getSession()
  if (!session.userId) redirect('/login')

  const [profileResult, balanceResult, historyResult, activeSessionResult] =
    await Promise.allSettled([
      getUserProfile(session.userId, session.accessToken),
      getWalletBalance(session.userId, session.accessToken),
      getWalletHistory(session.userId, session.accessToken),
      getActiveSession(session.userId, session.accessToken),
    ])

  const profile = profileResult.status === 'fulfilled' ? profileResult.value : null
  const balance = balanceResult.status === 'fulfilled' ? balanceResult.value.balance_nc : null
  const history = historyResult.status === 'fulfilled' ? historyResult.value.events : []
  const liveSession = activeSessionResult.status === 'fulfilled' ? activeSessionResult.value : null

  // XP/level from backend (profiles service is authoritative)
  const globalXP = profile?.global_xp ?? 0
  const level = profile?.global_level ?? 1
  const xpInLevel = globalXP % XP_PER_LEVEL
  const xpPct = Math.round((xpInLevel / XP_PER_LEVEL) * 100)

  const totalTopupNC = history
    .filter((e) => e.event_type === 'topup_confirmed')
    .reduce((sum, e) => sum + e.amount_nc, 0)

  const memberSince = profile?.created_at
    ? new Date(profile.created_at).toLocaleDateString('de-CH', {
        month: 'long',
        year: 'numeric',
      })
    : null

  return (
    <div className="flex flex-col min-h-screen">
      <NavBar displayName={session.displayName} balance={balance} />

      <main className="flex-1 max-w-lg mx-auto w-full px-4 py-6 space-y-5">
        {/* Identity card */}
        <div className="card">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-xl font-bold">{session.displayName}</p>
              {profile?.email && (
                <p className="text-sm text-nite-muted mt-0.5">{profile.email}</p>
              )}
              {memberSince && (
                <p className="text-xs text-nite-muted mt-1">Member since {memberSince}</p>
              )}
            </div>
            <div className="text-right shrink-0">
              <span className="inline-block bg-nite-accent/10 border border-nite-accent/30 text-nite-accent text-xs font-bold px-2 py-1 rounded-lg">
                LEVEL {level}
              </span>
            </div>
          </div>

          {/* XP bar */}
          <div className="mt-4">
            <div className="flex justify-between text-xs text-nite-muted mb-1">
              <span>XP: {xpInLevel} / {XP_PER_LEVEL}</span>
              <span>Level {level + 1} in {XP_PER_LEVEL - xpInLevel} XP</span>
            </div>
            <div className="h-2 rounded-full bg-nite-border overflow-hidden">
              <div
                className="h-full bg-nite-accent rounded-full transition-all"
                style={{ width: `${xpPct}%` }}
              />
            </div>
          </div>
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-3 gap-3">
          <div className="card text-center py-4">
            <p className="text-2xl font-black text-nite-accent">
              {balance !== null ? balance.toLocaleString() : '—'}
            </p>
            <p className="text-xs text-nite-muted mt-1">NC Balance</p>
          </div>
          <div className="card text-center py-4">
            <p className="text-2xl font-black">
              {history.filter((e) => e.event_type === 'venue_checkin').length || '—'}
            </p>
            <p className="text-xs text-nite-muted mt-1">Visits</p>
          </div>
          <div className="card text-center py-4">
            <p className="text-2xl font-black">
              {totalTopupNC > 0 ? `${totalTopupNC}` : '—'}
            </p>
            <p className="text-xs text-nite-muted mt-1">NC Loaded</p>
          </div>
        </div>

        {/* Active session */}
        {liveSession && (
          <div className="card border-nite-accent/30 bg-amber-950/10">
            <p className="text-xs text-nite-accent font-semibold uppercase tracking-wider mb-1">
              Currently checked in
            </p>
            {liveSession.nitetap_uid && (
              <p className="text-xs font-mono text-nite-muted">
                NiteTap: {liveSession.nitetap_uid}
              </p>
            )}
            <p className="text-sm mt-1">
              Session spend:{' '}
              <span className="text-nite-accent font-semibold">
                {liveSession.total_spend_nc.toLocaleString()} NC
              </span>
            </p>
            <Link href="/session" className="inline-block mt-2 text-sm text-nite-accent hover:underline">
              View session →
            </Link>
          </div>
        )}

        {/* NiteTap info */}
        <div className="card">
          <p className="text-xs text-nite-muted uppercase tracking-wider font-semibold mb-2">
            NiteTap
          </p>
          {liveSession?.nitetap_uid ? (
            <p className="font-mono text-sm text-nite-text">{liveSession.nitetap_uid}</p>
          ) : (
            <p className="text-sm text-nite-muted">
              No NiteTap linked. Tap your wristband at a venue entrance — it will appear here once linked to your account.
            </p>
          )}
        </div>

        {/* Activity history */}
        <section>
          <h2 className="text-xs font-semibold text-nite-muted uppercase tracking-wider mb-3">
            Activity
          </h2>

          {historyResult.status === 'rejected' ? (
            <p className="text-xs text-nite-muted card text-center py-4">
              Activity history coming soon.
            </p>
          ) : history.length === 0 ? (
            <p className="text-xs text-nite-muted card text-center py-4">
              No activity yet — top up and head to a venue!
            </p>
          ) : (
            <div className="space-y-2">
              {history.slice(0, 20).map((e) => (
                <div
                  key={e.event_id}
                  className="flex items-center justify-between gap-3 card py-3"
                >
                  <div className="flex items-center gap-2.5">
                    <span className="text-base">{eventIcon(e.event_type)}</span>
                    <div>
                      <p className="text-sm font-medium">{eventLabel(e.event_type)}</p>
                      <p className="text-xs text-nite-muted">
                        {new Date(e.occurred_at).toLocaleDateString('de-CH', {
                          day: 'numeric',
                          month: 'short',
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </p>
                    </div>
                  </div>
                  <span
                    className={`text-sm font-semibold shrink-0 ${
                      e.amount_nc >= 0 ? 'text-green-400' : 'text-nite-text'
                    }`}
                  >
                    {e.amount_nc >= 0 ? '+' : ''}
                    {e.amount_nc.toLocaleString()} NC
                  </span>
                </div>
              ))}
            </div>
          )}
        </section>

        <Link href="/" className="block text-center text-sm text-nite-muted hover:text-nite-text">
          ← Home
        </Link>
      </main>
    </div>
  )
}

function eventIcon(type: string): string {
  switch (type) {
    case 'topup_confirmed': return '💳'
    case 'order_paid': return '🍺'
    case 'venue_checkin': return '🏟️'
    case 'session_closed': return '👋'
    case 'refund_created': return '↩️'
    default: return '•'
  }
}

function eventLabel(type: string): string {
  switch (type) {
    case 'topup_confirmed': return 'Top-up'
    case 'order_paid': return 'Purchase'
    case 'venue_checkin': return 'Check-in'
    case 'session_closed': return 'Session ended'
    case 'refund_created': return 'Refund'
    default: return type.replace(/_/g, ' ')
  }
}
