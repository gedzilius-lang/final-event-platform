import { getSession } from '@/lib/session'
import { backendFetch, type Venue, type CatalogItem } from '@/lib/api'
import Link from 'next/link'

export default async function VenueDetailPage({
  params,
}: {
  params: { venue_id: string }
}) {
  const session = await getSession()
  const { venue_id } = params

  let venue: Venue | null = null
  let items: CatalogItem[] = []
  let err = ''

  try {
    venue = await backendFetch<Venue>(`/catalog/venues/${venue_id}`, session.accessToken)
  } catch (e: unknown) { err = (e as Error).message }

  if (venue) {
    try {
      const data = await backendFetch<{ items: CatalogItem[] }>(
        `/catalog/venues/${venue_id}/items`,
        session.accessToken,
      )
      items = data.items ?? []
    } catch {
      // non-fatal
    }
  }

  return (
    <div className="p-8">
      <div className="mb-6">
        <Link href="/admin/catalog/venues" className="text-sm text-gray-400 hover:text-white">← Venues</Link>
      </div>

      {err && <div className="mb-4 rounded-lg bg-red-950/40 border border-red-800 px-4 py-3 text-sm text-red-300">{err}</div>}

      {venue && (
        <>
          <div className="flex items-start justify-between mb-8">
            <div>
              <h1 className="text-2xl font-bold text-white">{venue.name}</h1>
              <p className="text-sm text-gray-400 mt-1">
                <span className="font-mono bg-gray-900 px-1.5 py-0.5 rounded text-xs">{venue.slug}</span>
                {' · '}{venue.city}{' · '}Capacity: {venue.capacity}
              </p>
            </div>
            <span className={`text-xs px-2 py-1 rounded-full ${venue.is_active ? 'bg-green-900/40 text-green-400' : 'bg-gray-800 text-gray-500'}`}>
              {venue.is_active ? 'active' : 'inactive'}
            </span>
          </div>

          <div className="grid grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
            {[
              { label: 'Address', value: venue.address || '—' },
              { label: 'Timezone', value: venue.timezone },
              { label: 'Venue ID', value: venue.venue_id, mono: true },
            ].map(f => (
              <div key={f.label} className="bg-gray-900 border border-gray-800 rounded-xl px-4 py-3">
                <p className="text-xs text-gray-500 mb-1">{f.label}</p>
                <p className={`text-sm text-white ${f.mono ? 'font-mono text-xs' : ''}`}>{f.value}</p>
              </div>
            ))}
          </div>

          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-white">Catalog Items ({items.length})</h2>
            <Link href={`/admin/catalog/items/new?venue_id=${venue_id}`}
              className="rounded-lg bg-brand-500 hover:bg-brand-600 px-4 py-2 text-sm font-medium text-white">
              + Add item
            </Link>
          </div>

          <div className="rounded-xl border border-gray-800 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-900 text-gray-400">
                <tr>
                  <th className="text-left px-4 py-3 font-medium">Icon</th>
                  <th className="text-left px-4 py-3 font-medium">Name</th>
                  <th className="text-left px-4 py-3 font-medium">Category</th>
                  <th className="text-right px-4 py-3 font-medium">Price (CHF)</th>
                  <th className="text-left px-4 py-3 font-medium">Status</th>
                  <th className="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800">
                {items.length === 0 && (
                  <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-500">No items yet.</td></tr>
                )}
                {items.map(item => (
                  <tr key={item.item_id} className="bg-gray-950 hover:bg-gray-900">
                    <td className="px-4 py-3 text-xl">{item.icon ?? '—'}</td>
                    <td className="px-4 py-3 text-white font-medium">{item.name}</td>
                    <td className="px-4 py-3 text-gray-400 capitalize">{item.category}</td>
                    <td className="px-4 py-3 text-right text-gray-200">{(item.price_nc / 100).toFixed(2)}</td>
                    <td className="px-4 py-3">
                      <span className={`text-xs px-2 py-1 rounded-full ${item.is_active ? 'bg-green-900/40 text-green-400' : 'bg-gray-800 text-gray-500'}`}>
                        {item.is_active ? 'active' : 'inactive'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <Link href={`/admin/catalog/items/${item.item_id}?venue_id=${venue_id}`}
                        className="text-xs text-brand-100 hover:text-white">Edit →</Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
    </div>
  )
}
