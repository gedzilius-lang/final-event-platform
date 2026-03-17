'use client'
// Security guest lookup panel.
import { useState } from 'react'

interface GuestProfile {
  user_id: string
  display_name: string
  email?: string
  role: string
  created_at?: string
}

interface ActiveSession {
  session_id: string
  venue_id: string
  opened_at: string
  total_spend_nc: number
  status: string
  nitetap_uid?: string
}

interface LookupResult {
  profile: GuestProfile
  active_session: ActiveSession | null
}

export default function SecurityPanel() {
  const [query, setQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<LookupResult | null>(null)
  const [error, setError] = useState('')

  async function lookup(e: React.FormEvent) {
    e.preventDefault()
    if (!query.trim()) return
    setError('')
    setResult(null)
    setLoading(true)
    try {
      const res = await fetch(
        `/api/staff/guest-lookup?email=${encodeURIComponent(query.trim())}`,
      )
      if (!res.ok) {
        const d = await res.json()
        setError(d.error ?? 'Not found')
        return
      }
      const profile: GuestProfile = await res.json()

      // Also fetch active session
      let active_session: ActiveSession | null = null
      try {
        const sessRes = await fetch(`/api/staff/guest-session?user_id=${profile.user_id}`)
        if (sessRes.ok) active_session = await sessRes.json()
      } catch { /* non-fatal */ }

      setResult({ profile, active_session })
    } catch {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-4">
      <form onSubmit={lookup} className="flex gap-2">
        <input
          type="email"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="input flex-1"
          placeholder="Guest email"
          autoComplete="off"
        />
        <button type="submit" disabled={loading} className="btn-primary px-4">
          {loading ? '…' : 'Search'}
        </button>
      </form>

      {error && (
        <div className="card border-red-900/40 bg-red-950/10">
          <p className="text-red-400 text-sm">{error}</p>
        </div>
      )}

      {result && (
        <div className="space-y-3">
          {/* Profile card */}
          <div className="card">
            <div className="flex items-start justify-between">
              <div>
                <p className="font-bold">{result.profile.display_name}</p>
                <p className="text-xs text-nite-muted">{result.profile.email}</p>
                {result.profile.created_at && (
                  <p className="text-xs text-nite-muted mt-0.5">
                    Member since{' '}
                    {new Date(result.profile.created_at).toLocaleDateString('de-CH', {
                      day: 'numeric',
                      month: 'short',
                      year: 'numeric',
                    })}
                  </p>
                )}
              </div>
              <span className="text-xs px-2 py-0.5 rounded-full bg-nite-border text-nite-muted capitalize">
                {result.profile.role.replace('_', ' ')}
              </span>
            </div>
            <p className="text-xs font-mono text-nite-muted mt-2 break-all">
              {result.profile.user_id}
            </p>
          </div>

          {/* Session status */}
          {result.active_session ? (
            <div className="card border-nite-accent/30 bg-amber-950/10">
              <p className="text-xs text-nite-accent font-semibold uppercase tracking-wider mb-1">
                Currently checked in
              </p>
              <p className="text-sm">
                In since{' '}
                {new Date(result.active_session.opened_at).toLocaleTimeString('de-CH', {
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </p>
              <p className="text-sm text-nite-accent font-semibold">
                Spend: {result.active_session.total_spend_nc} NC
              </p>
              {result.active_session.nitetap_uid && (
                <p className="text-xs font-mono text-nite-muted mt-1">
                  UID: {result.active_session.nitetap_uid}
                </p>
              )}
            </div>
          ) : (
            <div className="card">
              <p className="text-sm text-nite-muted">Not currently checked in.</p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
