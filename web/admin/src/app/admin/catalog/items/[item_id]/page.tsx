'use client'
import { useState, useEffect, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

interface Item {
  item_id: string
  venue_id: string
  name: string
  category: string
  price_nc: number
  icon?: string
  happy_hour_price_nc?: number
  display_order: number
  is_active: boolean
}

function ItemDetailContent({ itemId }: { itemId: string }) {
  const router = useRouter()
  const searchParams = useSearchParams()
  const venueId = searchParams.get('venue_id') ?? ''

  const [item, setItem] = useState<Item | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    if (!venueId) { setError('venue_id missing from URL'); setLoading(false); return }
    fetch(`/api/catalog/venues/${venueId}/items/${itemId}`)
      .then(r => r.json())
      .then(d => { setItem(d); setLoading(false) })
      .catch(e => { setError(e.message); setLoading(false) })
  }, [itemId, venueId])

  async function handleDelete() {
    if (!confirm(`Delete "${item?.name}"? This cannot be undone.`)) return
    setDeleting(true)
    try {
      const res = await fetch(`/api/catalog/venues/${venueId}/items/${itemId}`, { method: 'DELETE' })
      if (!res.ok) { setError((await res.json()).error ?? 'Delete failed'); return }
      router.push(`/admin/catalog/items?venue_id=${venueId}`)
      router.refresh()
    } finally { setDeleting(false) }
  }

  const back = venueId
    ? `/admin/catalog/venues/${venueId}`
    : '/admin/catalog/items'

  return (
    <div className="p-8 max-w-lg">
      <a href={back} className="text-sm text-gray-400 hover:text-white">← Back</a>

      {loading && <p className="mt-6 text-sm text-gray-400">Loading…</p>}
      {error && <div className="mt-6 rounded-lg bg-red-950/40 border border-red-800 px-4 py-3 text-sm text-red-300">{error}</div>}

      {item && (
        <>
          <div className="mt-6 mb-8 flex items-center gap-4">
            <span className="text-4xl">{item.icon ?? '🍹'}</span>
            <div>
              <h1 className="text-2xl font-bold text-white">{item.name}</h1>
              <p className="text-sm text-gray-400 capitalize">{item.category}</p>
            </div>
          </div>

          <div className="space-y-3 mb-8">
            {[
              { label: 'Price', value: `CHF ${(item.price_nc / 100).toFixed(2)}` },
              { label: 'Happy Hour Price', value: item.happy_hour_price_nc != null ? `CHF ${(item.happy_hour_price_nc / 100).toFixed(2)}` : '—' },
              { label: 'Display Order', value: item.display_order },
              { label: 'Status', value: item.is_active ? 'Active' : 'Inactive' },
              { label: 'Item ID', value: item.item_id, mono: true },
            ].map(f => (
              <div key={f.label} className="flex justify-between py-2 border-b border-gray-800">
                <span className="text-sm text-gray-400">{f.label}</span>
                <span className={`text-sm text-white ${(f as { mono?: boolean }).mono ? 'font-mono text-xs' : ''}`}>{f.value}</span>
              </div>
            ))}
          </div>

          <div className="bg-gray-900 border border-gray-800 rounded-xl px-4 py-4">
            <p className="text-xs text-gray-500 mb-3">Item editing is coming in M4.1. For now you can delete and re-create.</p>
            <button
              onClick={handleDelete}
              disabled={deleting}
              className="rounded-lg bg-red-900/60 hover:bg-red-800 border border-red-700 disabled:opacity-50 px-4 py-2 text-sm font-medium text-red-200"
            >
              {deleting ? 'Deleting…' : 'Delete item'}
            </button>
          </div>
        </>
      )}
    </div>
  )
}

export default function ItemDetailPage({ params }: { params: { item_id: string } }) {
  return <Suspense><ItemDetailContent itemId={params.item_id} /></Suspense>
}
