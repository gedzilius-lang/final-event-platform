// Manager / venue_admin overview — live session count, today's spend, quick actions.
import { redirect } from 'next/navigation'
import Link from 'next/link'
import { requireSession } from '@/lib/session'
import { listActiveSessions, backendFetch, ApiError } from '@/lib/api'

interface RevenueSummary {
  venue_id: string
  total_orders_nc: number
  total_topups_nc: number
  order_count: number
  session_count: number
}

export default async function ManagerPage() {
  const session = await requireSession()
  if (!['venue_admin', 'nitecore'].includes(session.role)) redirect('/staff')

  if (!session.venueId) {
    return (
      <div className="card text-center py-10">
        <p className="text-nite-muted text-sm">No venue assigned to your account.</p>
      </div>
    )
  }

  const now = new Date()
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate()).toISOString()
  const todayEnd = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1).toISOString()

  const [sessionsResult, revenueResult] = await Promise.allSettled([
    listActiveSessions(session.venueId, session.accessToken),
    backendFetch<RevenueSummary>(
      `/reporting/venues/${session.venueId}/revenue?from=${todayStart}&to=${todayEnd}`,
      session.accessToken,
    ),
  ])

  const activeSessions =
    sessionsResult.status === 'fulfilled' ? sessionsResult.value : { sessions: [], count: 0 }
  const revenue = revenueResult.status === 'fulfilled' ? revenueResult.value : null

  return (
    <div className="space-y-5">
      <h1 className="text-sm font-semibold uppercase tracking-wider text-nite-muted">
        Live Overview
      </h1>

      {/* Stats grid */}
      <div className="grid grid-cols-2 gap-3">
        <div className="card text-center py-5">
          <p className="text-4xl font-black text-nite-accent">{activeSessions.count}</p>
          <p className="text-xs text-nite-muted mt-1 uppercase tracking-wider">Checked in</p>
        </div>
        <div className="card text-center py-5">
          <p className="text-4xl font-black">{revenue?.order_count ?? '—'}</p>
          <p className="text-xs text-nite-muted mt-1 uppercase tracking-wider">Orders</p>
        </div>
        <div className="card text-center py-5">
          <p className="text-3xl font-black text-nite-accent">
            {revenue ? `${revenue.total_orders_nc.toLocaleString()} NC` : '—'}
          </p>
          <p className="text-xs text-nite-muted mt-1 uppercase tracking-wider">Revenue</p>
        </div>
        <div className="card text-center py-5">
          <p className="text-3xl font-black">
            {revenue ? `${revenue.total_topups_nc.toLocaleString()} NC` : '—'}
          </p>
          <p className="text-xs text-nite-muted mt-1 uppercase tracking-wider">Top-ups</p>
        </div>
      </div>

      {/* Active sessions list */}
      {activeSessions.sessions.length > 0 && (
        <section>
          <h2 className="text-xs font-semibold text-nite-muted uppercase tracking-wider mb-3">
            Active sessions ({activeSessions.count})
          </h2>
          <div className="space-y-2">
            {activeSessions.sessions.slice(0, 20).map((s) => (
              <div
                key={s.session_id}
                className="card flex items-center justify-between gap-3 py-3"
              >
                <div>
                  <p className="text-xs font-mono text-nite-muted">
                    {s.nitetap_uid ?? s.session_id.slice(0, 8) + '…'}
                  </p>
                  <p className="text-xs text-nite-muted">
                    In{' '}
                    {new Date(s.opened_at).toLocaleTimeString('de-CH', {
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </p>
                </div>
                <span className="text-sm font-semibold text-nite-accent shrink-0">
                  {s.total_spend_nc} NC
                </span>
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Quick links */}
      <section className="grid grid-cols-2 gap-3">
        <Link
          href="/staff/door"
          className="card text-center py-5 hover:border-nite-accent/40 transition-colors"
        >
          <span className="text-2xl">🚪</span>
          <p className="text-xs font-medium mt-1">Door</p>
        </Link>
        <Link
          href="/staff/bar"
          className="card text-center py-5 hover:border-nite-accent/40 transition-colors"
        >
          <span className="text-2xl">🍺</span>
          <p className="text-xs font-medium mt-1">Bar POS</p>
        </Link>
        <Link
          href="/staff/security"
          className="card text-center py-5 hover:border-nite-accent/40 transition-colors"
        >
          <span className="text-2xl">🔍</span>
          <p className="text-xs font-medium mt-1">Security</p>
        </Link>
        <a
          href="https://admin.peoplewelike.club"
          target="_blank"
          rel="noopener noreferrer"
          className="card text-center py-5 hover:border-nite-accent/40 transition-colors"
        >
          <span className="text-2xl">⚙️</span>
          <p className="text-xs font-medium mt-1">Admin</p>
        </a>
      </section>
    </div>
  )
}
