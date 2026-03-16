import { getSession } from '@/lib/session'
import { backendFetch, type CatalogItem } from '@/lib/api'
import Link from 'next/link'

export default async function CatalogItemsPage({
  searchParams,
}: {
  searchParams: { venue_id?: string }
}) {
  const session = await getSession()
  const venueId = searchParams.venue_id ?? session.venueId ?? ''

  let items: CatalogItem[] = []
  let err = ''
  if (venueId) {
    try {
      const data = await backendFetch<{ items: CatalogItem[] }>(
        `/catalog/venues/${venueId}/items`, session.accessToken,
      )
      items = data.items ?? []
    } catch (e: unknown) { err = (e as Error).message }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-white">Catalog Items</h1>
        {venueId && (
          <Link href={`/admin/catalog/items/new?venue_id=${venueId}`}
            className="rounded-lg bg-brand-500 hover:bg-brand-600 px-4 py-2 text-sm font-medium text-white">
            + Add item
          </Link>
        )}
      </div>
      {!venueId && (
        <p className="text-sm text-gray-400 mb-4">
          Select a venue: <Link href="/admin/catalog/venues" className="text-brand-100 underline">Venues</Link>
        </p>
      )}
      {err && <div className="mb-4 rounded-lg bg-red-950/40 border border-red-800 px-4 py-3 text-sm text-red-300">{err}</div>}
      <div className="rounded-xl border border-gray-800 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-900 text-gray-400">
            <tr>
              <th className="text-left px-4 py-3 font-medium">Icon</th>
              <th className="text-left px-4 py-3 font-medium">Name</th>
              <th className="text-left px-4 py-3 font-medium">Category</th>
              <th className="text-right px-4 py-3 font-medium">Price (CHF)</th>
              <th className="text-right px-4 py-3 font-medium">Happy Hour</th>
              <th className="text-left px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-800">
            {items.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-500">{venueId ? 'No items yet.' : ''}</td></tr>
            )}
            {items.map(item => (
              <tr key={item.item_id} className="bg-gray-950 hover:bg-gray-900">
                <td className="px-4 py-3 text-xl">{item.icon ?? '—'}</td>
                <td className="px-4 py-3 text-white font-medium">{item.name}</td>
                <td className="px-4 py-3 text-gray-400 capitalize">{item.category}</td>
                <td className="px-4 py-3 text-right text-gray-200">{(item.price_nc / 100).toFixed(2)}</td>
                <td className="px-4 py-3 text-right text-gray-400">
                  {item.happy_hour_price_nc != null ? (item.happy_hour_price_nc / 100).toFixed(2) : '—'}
                </td>
                <td className="px-4 py-3">
                  <span className={`text-xs px-2 py-1 rounded-full ${item.is_active ? 'bg-green-900/40 text-green-400' : 'bg-gray-800 text-gray-500'}`}>
                    {item.is_active ? 'active' : 'inactive'}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <Link href={`/admin/catalog/items/${item.item_id}?venue_id=${venueId}`}
                    className="text-xs text-brand-100 hover:text-white">Edit →</Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
