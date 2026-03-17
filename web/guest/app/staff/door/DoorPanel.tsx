'use client'
// Door staff check-in panel — client component.
// Looks up a guest by email, then opens a venue session.
import { useState } from 'react'

interface Props {
  venueId: string
}

interface GuestInfo {
  user_id: string
  display_name: string
  email?: string
  role: string
}

interface SessionInfo {
  session_id: string
  status: string
  total_spend_nc: number
  opened_at: string
}

type Step = 'lookup' | 'confirm' | 'checked_in'

export default function DoorPanel({ venueId }: Props) {
  const [step, setStep] = useState<Step>('lookup')
  const [email, setEmail] = useState('')
  const [nfcUid, setNfcUid] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [guest, setGuest] = useState<GuestInfo | null>(null)
  const [session, setSession] = useState<SessionInfo | null>(null)

  async function lookupGuest(e: React.FormEvent) {
    e.preventDefault()
    if (!email.trim()) return
    setError('')
    setLoading(true)
    try {
      const res = await fetch(`/api/staff/guest-lookup?email=${encodeURIComponent(email.trim())}`)
      if (!res.ok) {
        const d = await res.json()
        setError(d.error ?? 'Guest not found')
        return
      }
      const data: GuestInfo = await res.json()
      setGuest(data)
      setStep('confirm')
    } catch {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  async function checkin() {
    if (!guest) return
    setError('')
    setLoading(true)
    try {
      const res = await fetch('/api/staff/checkin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          user_id: guest.user_id,
          venue_id: venueId,
          nfc_uid: nfcUid.trim() || undefined,
        }),
      })
      if (!res.ok) {
        const d = await res.json()
        setError(d.error ?? 'Check-in failed')
        return
      }
      const sess: SessionInfo = await res.json()
      setSession(sess)
      setStep('checked_in')
    } catch {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  function reset() {
    setStep('lookup')
    setEmail('')
    setNfcUid('')
    setGuest(null)
    setSession(null)
    setError('')
  }

  return (
    <div className="card space-y-4">
      <h2 className="text-sm font-semibold uppercase tracking-wider text-nite-muted">
        Check In Guest
      </h2>

      {step === 'lookup' && (
        <form onSubmit={lookupGuest} className="space-y-3">
          <div>
            <label className="label">Guest email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="input"
              placeholder="guest@example.com"
              required
              autoComplete="off"
            />
          </div>
          <button type="submit" disabled={loading} className="btn-primary w-full">
            {loading ? 'Looking up…' : 'Look up guest'}
          </button>
        </form>
      )}

      {step === 'confirm' && guest && (
        <div className="space-y-4">
          <div className="bg-nite-bg border border-nite-border rounded-lg p-3">
            <p className="font-semibold">{guest.display_name}</p>
            <p className="text-xs text-nite-muted">{guest.email}</p>
          </div>
          <div>
            <label className="label">NiteTap UID (optional)</label>
            <input
              type="text"
              value={nfcUid}
              onChange={(e) => setNfcUid(e.target.value)}
              className="input font-mono"
              placeholder="Leave blank for QR check-in"
              autoComplete="off"
            />
          </div>
          <div className="flex gap-2">
            <button onClick={reset} className="btn-ghost flex-1">
              Back
            </button>
            <button onClick={checkin} disabled={loading} className="btn-primary flex-1">
              {loading ? 'Checking in…' : '✓ Check in'}
            </button>
          </div>
        </div>
      )}

      {step === 'checked_in' && session && guest && (
        <div className="space-y-3">
          <div className="text-center py-4">
            <span className="text-4xl">✓</span>
            <p className="font-bold mt-2">{guest.display_name}</p>
            <p className="text-sm text-nite-muted">Checked in successfully</p>
            <p className="text-xs font-mono text-nite-muted mt-1">
              Session: {session.session_id.slice(0, 8)}…
            </p>
          </div>
          <button onClick={reset} className="btn-primary w-full">
            Next guest
          </button>
        </div>
      )}

      {error && (
        <p className="text-red-400 text-sm text-center">{error}</p>
      )}
    </div>
  )
}
