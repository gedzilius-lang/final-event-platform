import { getSession } from '@/lib/session'
import { backendFetch, type VenueSession } from '@/lib/api'

export default async function SessionsPage() {
  const session = await getSession()
  const venueId = session.venueId ?? ''
  let sessions: VenueSession[] = []
  let err = ''
  if (venueId) {
    try {
      const data = await backendFetch<{ sessions: VenueSession[] }>(
        `/sessions/venues/${venueId}/active`, session.accessToken,
      )
      sessions = data.sessions ?? []
    } catch (e: unknown) { err = (e as Error).message }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-white">Live Sessions</h1>
          <p className="text-sm text-gray-400 mt-1">
            {sessions.length} guest{sessions.length !== 1 ? 's' : ''} currently inside
          </p>
        </div>
      </div>
      {!venueId && <p className="text-sm text-gray-400">No venue associated with this account.</p>}
      {err && <div className="mb-4 rounded-lg bg-red-950/40 border border-red-800 px-4 py-3 text-sm text-red-300">{err}</div>}
      <div className="rounded-xl border border-gray-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-900 text-gray-400">
            <tr>
              <th className="text-left px-4 py-3 font-medium">Session</th>
              <th className="text-left px-4 py-3 font-medium">NiteTap</th>
              <th className="text-right px-4 py-3 font-medium">Spend (CHF)</th>
              <th className="text-left px-4 py-3 font-medium">Opened</th>
              <th className="text-left px-4 py-3 font-medium">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {sessions.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">{venueId ? 'No active sessions.' : ''}</td></tr>
            )}
            {sessions.map(s => (
              <tr key={s.session_id} className="bg-gray-950 hover:bg-gray-900">
                <td className="px-4 py-3 text-gray-300 font-mono text-xs">{s.session_id.substring(0, 8)}…</td>
                <td className="px-4 py-3 text-gray-400 font-mono text-xs">{s.nitetap_uid ?? '—'}</td>
                <td className="px-4 py-3 text-right text-white">{(s.total_spend_nc / 100).toFixed(2)}</td>
                <td className="px-4 py-3 text-gray-400 text-xs">{new Date(s.opened_at).toLocaleTimeString()}</td>
                <td className="px-4 py-3">
                  <span className={`text-xs px-2 py-1 rounded-full ${s.status === 'open' ? 'bg-green-900/40 text-green-400' : 'bg-gray-800 text-gray-500'}`}>
                    {s.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
