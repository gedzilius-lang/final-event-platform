// Door staff surface — guest check-in, active session list.
import { redirect } from 'next/navigation'
import { requireSession } from '@/lib/session'
import { listActiveSessions, ApiError } from '@/lib/api'
import DoorPanel from './DoorPanel'

export default async function DoorPage() {
  const session = await requireSession()
  if (!session.venueId) {
    return (
      <div className="card text-center py-10">
        <p className="text-nite-muted text-sm">
          No venue assigned to your account. Ask your manager to assign a venue.
        </p>
      </div>
    )
  }

  const sessionsResult = await listActiveSessions(session.venueId, session.accessToken)
    .catch((e: ApiError) => {
      if (e.status === 403) return null // role not permitted (shouldn't happen)
      return { sessions: [], count: 0 }
    })

  return (
    <div className="space-y-5">
      {/* Active count */}
      <div className="grid grid-cols-2 gap-3">
        <div className="card text-center py-5">
          <p className="text-4xl font-black text-nite-accent">
            {sessionsResult?.count ?? '—'}
          </p>
          <p className="text-xs text-nite-muted mt-1 uppercase tracking-wider">Active</p>
        </div>
        <div className="card text-center py-5">
          <p className="text-4xl font-black">—</p>
          <p className="text-xs text-nite-muted mt-1 uppercase tracking-wider">Capacity</p>
        </div>
      </div>

      {/* Check-in panel */}
      <DoorPanel venueId={session.venueId} />

      {/* Active session list */}
      {sessionsResult && sessionsResult.sessions.length > 0 && (
        <section>
          <h2 className="text-xs font-semibold text-nite-muted uppercase tracking-wider mb-3">
            Checked in
          </h2>
          <div className="space-y-2">
            {sessionsResult.sessions.map((s) => (
              <div key={s.session_id} className="card flex items-center justify-between gap-3 py-3">
                <div>
                  <p className="text-sm font-mono text-nite-muted">
                    {s.nitetap_uid ?? 'Anonymous'}
                  </p>
                  <p className="text-xs text-nite-muted">
                    In at{' '}
                    {new Date(s.opened_at).toLocaleTimeString('de-CH', {
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </p>
                </div>
                <p className="text-sm text-nite-accent font-semibold shrink-0">
                  {s.total_spend_nc} NC
                </p>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  )
}
