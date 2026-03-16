'use client'
import { useState, FormEvent } from 'react'
import { useRouter } from 'next/navigation'

const ROLES = ['guest', 'venue_admin', 'bartender', 'door_staff', 'nitecore']

export default function AssignVenueButton({ userId, currentRole }: { userId: string; currentRole: string }) {
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setSaving(true); setError('')
    const form = new FormData(e.currentTarget)
    const payload = {
      venue_id: (form.get('venue_id') as string).trim(),
      role: form.get('role') as string,
    }
    try {
      const res = await fetch(`/api/admin/users/${userId}/venue`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) { setError((await res.json()).error ?? 'Failed'); return }
      setOpen(false)
      router.refresh()
    } finally { setSaving(false) }
  }

  if (!open) {
    return (
      <button onClick={() => setOpen(true)}
        className="text-xs text-gray-500 hover:text-brand-100 transition-colors">
        Assign venue
      </button>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-1.5 min-w-[200px]">
      <input name="venue_id" type="text" placeholder="venue_id (UUID or blank)"
        className="rounded bg-gray-800 border border-gray-700 px-2 py-1 text-xs text-white placeholder-gray-500 focus:outline-none focus:ring-1 focus:ring-brand-500" />
      <select name="role" defaultValue={currentRole}
        className="rounded bg-gray-800 border border-gray-700 px-2 py-1 text-xs text-white focus:outline-none focus:ring-1 focus:ring-brand-500">
        {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
      </select>
      {error && <p className="text-xs text-red-400">{error}</p>}
      <div className="flex gap-1">
        <button type="submit" disabled={saving}
          className="rounded bg-brand-500 hover:bg-brand-600 disabled:opacity-50 px-2 py-1 text-xs text-white">
          {saving ? '…' : 'Save'}
        </button>
        <button type="button" onClick={() => { setOpen(false); setError('') }}
          className="rounded bg-gray-700 hover:bg-gray-600 px-2 py-1 text-xs text-gray-300">
          Cancel
        </button>
      </div>
    </form>
  )
}
