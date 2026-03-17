// Home page — server component.
// Authenticated: shows wallet preview + venue feed + radio embed.
// Unauthenticated: shows landing / CTA.
import Link from 'next/link'
import { getSession } from '@/lib/session'
import { getWalletBalance, getVenues, getActiveSession } from '@/lib/api'
import NavBar from '@/components/NavBar'
import WalletBadge from '@/components/WalletBadge'
import RadioEmbed from '@/components/RadioEmbed'

export default async function Home() {
  const session = await getSession()
  const isLoggedIn = !!session.userId && !!session.accessToken

  if (!isLoggedIn) {
    return <LandingPage />
  }

  // Fetch in parallel — all errors are non-fatal
  const [walletResult, venuesResult, activeSessionResult] = await Promise.allSettled([
    getWalletBalance(session.userId, session.accessToken),
    getVenues(session.accessToken),
    getActiveSession(session.userId, session.accessToken),
  ])

  const balance =
    walletResult.status === 'fulfilled' ? walletResult.value.balance_nc : null
  const venues =
    venuesResult.status === 'fulfilled' ? venuesResult.value.venues : []
  const liveSession =
    activeSessionResult.status === 'fulfilled' ? activeSessionResult.value : null

  return (
    <div className="flex flex-col min-h-screen">
      <NavBar displayName={session.displayName} balance={balance} />

      <main className="flex-1 max-w-lg mx-auto w-full px-4 py-6 space-y-6">
        {/* Active venue session banner */}
        {liveSession && (
          <div className="card border-nite-accent bg-amber-950/20">
            <p className="text-xs text-nite-accent font-semibold uppercase tracking-wider mb-1">
              Active Session
            </p>
            <p className="font-semibold">You&apos;re checked in!</p>
            <p className="text-sm text-nite-muted mt-1">
              Session spend: {liveSession.total_spend_nc.toLocaleString()} NC
            </p>
            <Link href="/session" className="inline-block mt-3 btn-primary text-sm">
              View session →
            </Link>
          </div>
        )}

        {/* Wallet summary */}
        <WalletBadge balance={balance} />

        {/* Venue feed */}
        <section>
          <h2 className="text-lg font-bold mb-3">Tonight</h2>
          {venues.length === 0 ? (
            <p className="text-nite-muted text-sm">No venues listed yet.</p>
          ) : (
            <div className="space-y-3">
              {venues.map((v) => (
                <Link key={v.venue_id} href={`/venues/${v.slug}`} className="block">
                  <div className="card hover:border-nite-accent/50 transition-colors">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="font-semibold">{v.name}</p>
                        <p className="text-sm text-nite-muted">{v.city}</p>
                      </div>
                      <span
                        className={`text-xs px-2 py-0.5 rounded-full ${
                          v.is_active
                            ? 'bg-green-900/40 text-green-400'
                            : 'bg-nite-border text-nite-muted'
                        }`}
                      >
                        {v.is_active ? 'Open' : 'Closed'}
                      </span>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          )}
        </section>

        {/* Quick links */}
        <section className="grid grid-cols-2 gap-3">
          <Link href="/tickets" className="card hover:border-nite-accent/50 transition-colors text-center py-4">
            <span className="text-2xl">🎟️</span>
            <p className="text-sm font-medium mt-1">My Tickets</p>
          </Link>
          <Link href="/wallet" className="card hover:border-nite-accent/50 transition-colors text-center py-4">
            <span className="text-2xl">💰</span>
            <p className="text-sm font-medium mt-1">Top Up</p>
          </Link>
        </section>

        {/* Radio embed */}
        <RadioEmbed />
      </main>
    </div>
  )
}

function LandingPage() {
  return (
    <div className="flex flex-col min-h-screen">
      {/* Hero */}
      <div className="flex-1 flex flex-col items-center justify-center px-6 text-center py-16">
        <div className="mb-6">
          <span className="text-6xl">🌃</span>
        </div>
        <h1 className="text-4xl font-black mb-2 tracking-tight">
          People We Like
        </h1>
        <p className="text-nite-accent font-semibold mb-4">Powered by NiteOS</p>
        <p className="text-nite-muted text-sm max-w-xs mb-8 leading-relaxed">
          Tap. Pay. Enjoy. Cashless nights out — no card, no queue, no friction.
          Load NiteCoins, tap your wristband, and the venue handles the rest.
        </p>

        <div className="flex flex-col gap-3 w-full max-w-xs">
          <Link href="/login" className="btn-primary w-full text-center py-3 text-base">
            Sign in
          </Link>
          <Link href="/register" className="btn-ghost w-full text-center py-3 text-base">
            Create account
          </Link>
        </div>

        <p className="text-nite-muted text-xs mt-8">
          Already at a venue? Tap your NiteTap at the entrance.
        </p>
      </div>

      {/* Radio widget on landing page too */}
      <div className="max-w-lg mx-auto w-full px-4 pb-6">
        <RadioEmbed />
      </div>

      {/* Footer */}
      <footer className="py-5 text-center text-nite-muted text-xs border-t border-nite-border">
        <p>© {new Date().getFullYear()} People We Like · NiteOS platform</p>
      </footer>
    </div>
  )
}
