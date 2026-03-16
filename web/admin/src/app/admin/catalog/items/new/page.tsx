'use client'
import { useState, FormEvent, Suspense } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

function NewItemForm() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const venueId = searchParams.get('venue_id') ?? ''
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); setSaving(true); setError('')
    const form = new FormData(e.currentTarget)
    const hhPrice = form.get('happy_hour_price_chf') as string
    const payload = {
      venue_id: venueId,
      name: form.get('name'),
      category: form.get('category'),
      price_nc: Math.round(Number(form.get('price_chf')) * 100),
      icon: form.get('icon') || undefined,
      happy_hour_price_nc: hhPrice ? Math.round(Number(hhPrice) * 100) : undefined,
      display_order: Number(form.get('display_order')) || 0,
    }
    try {
      const res = await fetch(`/api/catalog/venues/${venueId}/items`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) { setError((await res.json()).error ?? 'Failed'); return }
      router.push(`/admin/catalog/items?venue_id=${venueId}`); router.refresh()
    } finally { setSaving(false) }
  }

  return (
    <div className="p-8 max-w-lg">
      <h1 className="text-2xl font-bold text-white mb-6">Add Catalog Item</h1>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Name</label>
          <input name="name" type="text" required
            className="w-full rounded-lg bg-gray-800 border border-gray-700 px-4 py-2.5 text-white focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Category</label>
          <select name="category" required
            className="w-full rounded-lg bg-gray-800 border border-gray-700 px-4 py-2.5 text-white focus:outline-none focus:ring-2 focus:ring-brand-500">
            <option value="drinks">Drinks</option>
            <option value="food">Food</option>
            <option value="entry">Entry</option>
            <option value="other">Other</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Price (CHF)</label>
          <input name="price_chf" type="number" step="0.01" min="0.01" required placeholder="8.00"
            className="w-full rounded-lg bg-gray-800 border border-gray-700 px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand-500" />
          <p className="mt-1 text-xs text-gray-500">1 CHF = 100 NC. Price stored in NC.</p>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Happy Hour Price (CHF, optional)</label>
          <input name="happy_hour_price_chf" type="number" step="0.01" min="0.01" placeholder="6.00"
            className="w-full rounded-lg bg-gray-800 border border-gray-700 px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Icon (emoji)</label>
          <input name="icon" type="text" maxLength={4} placeholder="🍺"
            className="w-full rounded-lg bg-gray-800 border border-gray-700 px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-1">Display order</label>
          <input name="display_order" type="number" min="0" placeholder="0"
            className="w-full rounded-lg bg-gray-800 border border-gray-700 px-4 py-2.5 text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-brand-500" />
        </div>
        {error && <p className="text-sm text-red-400 bg-red-950/40 border border-red-800 rounded px-3 py-2">{error}</p>}
        <div className="flex gap-3 pt-2">
          <button type="submit" disabled={saving}
            className="rounded-lg bg-brand-500 hover:bg-brand-600 disabled:opacity-50 px-5 py-2.5 text-sm font-semibold text-white">
            {saving ? 'Saving…' : 'Add item'}
          </button>
          <button type="button" onClick={() => router.back()}
            className="rounded-lg bg-gray-800 hover:bg-gray-700 px-5 py-2.5 text-sm text-gray-300">
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}

export default function NewItemPage() {
  return <Suspense><NewItemForm /></Suspense>
}
