// Active venue session page
import { redirect } from 'next/navigation'
import Link from 'next/link'
import { getSession } from '@/lib/session'
import { getActiveSession, getWalletBalance, ncToCHF } from '@/lib/api'
import NavBar from '@/components/NavBar'

export default async function SessionPage() {
  const session = await getSession()
  if (!session.userId) redirect('/login')

  const [sessionResult, balanceResult] = await Promise.allSettled([
    getActiveSession(session.userId, session.accessToken),
    getWalletBalance(session.userId, session.accessToken),
  ])

  const liveSession =
    sessionResult.status === 'fulfilled' ? sessionResult.value : null
  const balance =
    balanceResult.status === 'fulfilled' ? balanceResult.value.balance_nc : null

  return (
    <div className="flex flex-col min-h-screen">
      <NavBar displayName={session.displayName} balance={balance} />

      <main className="flex-1 max-w-lg mx-auto w-full px-4 py-6 space-y-6">
        <h1 className="text-2xl font-bold">Your Session</h1>

        {!liveSession ? (
          <div className="card text-center py-12">
            <span className="text-4xl">👋</span>
            <p className="text-nite-muted mt-3 mb-1">No active session.</p>
            <p className="text-sm text-nite-muted">
              Tap your NiteTap at a venue entrance to start one.
            </p>
            <Link href="/" className="inline-block mt-4 btn-primary text-sm">
              Browse venues
            </Link>
          </div>
        ) : (
          <>
            <div className="card border-nite-accent/30 bg-amber-950/10">
              <p className="text-xs text-nite-accent font-semibold uppercase tracking-wider mb-1">
                Checked in
              </p>
              <p className="text-nite-muted text-sm">
                Since{' '}
                {new Date(liveSession.opened_at).toLocaleTimeString('de-CH', {
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </p>
            </div>

            <div className="card text-center py-8">
              <p className="text-nite-muted text-sm mb-1">Session spend</p>
              <p className="text-5xl font-black text-nite-accent">
                {liveSession.total_spend_nc.toLocaleString()}
              </p>
              <p className="text-nite-muted text-sm mt-1">NC</p>
              <p className="text-nite-muted text-xs mt-0.5">
                ≈ CHF {ncToCHF(liveSession.total_spend_nc)}
              </p>
            </div>

            <div className="card">
              <p className="text-xs text-nite-muted uppercase tracking-wider mb-2">Wallet balance</p>
              <p className="text-2xl font-bold text-nite-text">
                {balance !== null ? `${balance.toLocaleString()} NC` : '—'}
              </p>
              {balance !== null && (
                <p className="text-xs text-nite-muted">≈ CHF {ncToCHF(balance)}</p>
              )}
              <Link href="/wallet" className="text-sm text-nite-accent mt-2 block hover:underline">
                Top up →
              </Link>
            </div>

            <div className="card">
              <p className="text-xs text-nite-muted uppercase tracking-wider mb-2">How to order</p>
              <p className="text-sm text-nite-muted">
                Walk to any NiteKiosk terminal, select your item, and tap your NiteTap to confirm.
                Your balance updates instantly.
              </p>
            </div>
          </>
        )}

        <Link href="/" className="block text-center text-sm text-nite-muted hover:text-nite-text">
          ← Home
        </Link>
      </main>
    </div>
  )
}
