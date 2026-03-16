import { getSession } from '@/lib/session'
import { backendFetch, type Venue } from '@/lib/api'
import Link from 'next/link'

export default async function VenuesPage() {
  const session = await getSession()
  let venues: Venue[] = []
  let err = ''
  try {
    const data = await backendFetch<{ venues: Venue[] }>('/catalog/venues', session.accessToken)
    venues = data.venues ?? []
  } catch (e: unknown) { err = (e as Error).message }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Venues</h1>
        {session.role === 'nitecore' && (
          <Link href="/admin/catalog/venues/new"
            className="rounded-lg bg-brand-500 hover:bg-brand-600 px-4 py-2 text-sm font-medium text-white">
            + New venue
          </Link>
        )}
      </div>
      {err && <div className="mb-4 rounded-lg bg-red-950/40 border border-red-800 px-4 py-3 text-sm text-red-300">{err}</div>}
      <div className="rounded-xl border border-gray-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-900 text-gray-400">
            <tr>
              <th className="text-left px-4 py-3 font-medium">Name</th>
              <th className="text-left px-4 py-3 font-medium">Slug</th>
              <th className="text-left px-4 py-3 font-medium">City</th>
              <th className="text-right px-4 py-3 font-medium">Capacity</th>
              <th className="text-left px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {venues.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No venues found.</td></tr>
            )}
            {venues.map(v => (
              <tr key={v.venue_id} className="bg-gray-950 hover:bg-gray-900 transition-colors">
                <td className="px-4 py-3 text-white font-medium">{v.name}</td>
                <td className="px-4 py-3 text-gray-400 font-mono text-xs">{v.slug}</td>
                <td className="px-4 py-3 text-gray-300">{v.city}</td>
                <td className="px-4 py-3 text-right text-gray-300">{v.capacity}</td>
                <td className="px-4 py-3">
                  <span className={`text-xs px-2 py-1 rounded-full ${v.is_active ? 'bg-green-900/40 text-green-400' : 'bg-gray-800 text-gray-500'}`}>
                    {v.is_active ? 'active' : 'inactive'}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <Link href={`/admin/catalog/venues/${v.venue_id}`} className="text-xs text-brand-100 hover:text-white">
                    Manage →
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
