import { getSession } from '@/lib/session'
import { backendFetch, type Device } from '@/lib/api'

export default async function DevicesPage() {
  const session = await getSession()
  const venueId = session.venueId ?? ''
  let devices: Device[] = []
  let err = ''
  if (venueId) {
    try {
      const data = await backendFetch<{ devices: Device[] }>(
        `/devices/venues/${venueId}`, session.accessToken,
      )
      devices = data.devices ?? []
    } catch (e: unknown) { err = (e as Error).message }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Devices</h1>
      </div>
      {!venueId && <p className="text-sm text-gray-400">No venue associated with this account.</p>}
      {err && <div className="mb-4 rounded-lg bg-red-950/40 border border-red-800 px-4 py-3 text-sm text-red-300">{err}</div>}
      <div className="rounded-xl border border-gray-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-900 text-gray-400">
            <tr>
              <th className="text-left px-4 py-3 font-medium">Name</th>
              <th className="text-left px-4 py-3 font-medium">Role</th>
              <th className="text-left px-4 py-3 font-medium">Status</th>
              <th className="text-left px-4 py-3 font-medium">Last heartbeat</th>
              <th className="text-left px-4 py-3 font-medium">IP</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {devices.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-gray-500">{venueId ? 'No devices enrolled yet.' : ''}</td></tr>
            )}
            {devices.map(d => (
              <tr key={d.device_id} className="bg-gray-950 hover:bg-gray-900">
                <td className="px-4 py-3 text-white">{d.device_name ?? '—'}</td>
                <td className="px-4 py-3 text-gray-400 capitalize">{d.device_role}</td>
                <td className="px-4 py-3">
                  <span className={`text-xs px-2 py-1 rounded-full ${
                    d.status === 'active' ? 'bg-green-900/40 text-green-400' :
                    d.status === 'pending' ? 'bg-yellow-900/40 text-yellow-400' :
                    'bg-gray-800 text-gray-500'
                  }`}>{d.status}</span>
                </td>
                <td className="px-4 py-3 text-gray-400 text-xs">
                  {d.last_heartbeat ? new Date(d.last_heartbeat).toLocaleString() : '—'}
                </td>
                <td className="px-4 py-3 text-gray-400 font-mono text-xs">{d.last_seen_ip ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
