'use client'

import { useState, FormEvent } from 'react'
import { useRouter } from 'next/navigation'

export default function NewVenuePage() {
  const router = useRouter()
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setSaving(true)
    setError('')
    const form = new FormData(e.currentTarget)
    const payload = {
      name:     form.get('name'),
      slug:     form.get('slug'),
      city:     form.get('city'),
      address:  form.get('address'),
      capacity: Number(form.get('capacity')),
      staff_pin: form.get('staff_pin'),
      timezone: form.get('timezone') || 'Europe/Zurich',
    }
    try {
      const res = await fetch('/api/catalog/venues', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const d = await res.json()
        setError(d.error ?? 'Failed to create venue')
        return
      }
      router.push('/catalog/venues')
      router.refresh()
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="p-8 max-w-xl">
      <h1 className="text-2xl font-bold text-white mb-6">New Venue</h1>
      <form onSubmit={handleSubmit} className="space-y-4">
        {[
          { name: 'name',      label: 'Venue name',    type: 'text',   required: true },
          { name: 'slug',      label: 'Slug',          type: 'text',   required: true,  placeholder: 'venue-alpha' },
          { name: 'city',      label: 'City',          type: 'text',   required: true,  placeholder: 'Zurich' },
          { name: 'address',   label: 'Address',       type: 'text',   required: false },
          { name: 'capacity',  label: 'Capacity',      type: 'number', required: true,  placeholder: '200' },
          { name: 'staff_pin', label: 'Staff PIN (4 digits)', type: 'text', required: true, placeholder: '1234' },
          { name: 'timezone',  label: 'Timezone',      type: 'text',   required: false, placeholder: 'Europe/Zurich' },
        ].map(f => (
          <div key={f.name}>
            <label className="block text-sm font-medium text-gray-300 mb-1">{f.label}</label>
            <input
              name={f.name}
              type={f.type}
              required={f.required}
              placeholder={f.placeholder}
              className="w-full rounded-lg bg-gray-800 border border-gray-700 px-4 py-2.5
                         text-white placeholder-gray-500 focus:outline-none focus:ring-2
                         focus:ring-brand-500"
            />
          </div>
        ))}

        {error && (
          <p className="text-sm text-red-400 bg-red-950/40 border border-red-800 rounded px-3 py-2">
            {error}
          </p>
        )}

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={saving}
            className="rounded-lg bg-brand-500 hover:bg-brand-600 disabled:opacity-50
                       px-5 py-2.5 text-sm font-semibold text-white"
          >
            {saving ? 'Creating…' : 'Create venue'}
          </button>
          <button
            type="button"
            onClick={() => router.back()}
            className="rounded-lg bg-gray-800 hover:bg-gray-700 px-5 py-2.5 text-sm text-gray-300"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}
