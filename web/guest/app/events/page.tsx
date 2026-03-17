// Events / Venue discovery page
import { redirect } from 'next/navigation'
import Link from 'next/link'
import { getSession } from '@/lib/session'
import { getVenues, getWalletBalance, VenueSummary } from '@/lib/api'
import NavBar from '@/components/NavBar'

export default async function EventsPage() {
  const session = await getSession()
  if (!session.userId) redirect('/login')

  const [venuesResult, balanceResult] = await Promise.allSettled([
    getVenues(session.accessToken),
    getWalletBalance(session.userId, session.accessToken),
  ])

  const venues =
    venuesResult.status === 'fulfilled' ? venuesResult.value.venues : []
  const fetchError = venuesResult.status === 'rejected' ? 'Failed to load venues' : null
  const balance =
    balanceResult.status === 'fulfilled' ? balanceResult.value.balance_nc : null

  return (
    <div className="flex flex-col min-h-screen">
      <NavBar displayName={session.displayName} balance={balance} />

      <main className="flex-1 max-w-lg mx-auto w-full px-4 py-6 space-y-6">
        <div>
          <h1 className="text-2xl font-bold">Venues</h1>
          <p className="text-sm text-nite-muted mt-1">Tap a venue to see the menu.</p>
        </div>

        {fetchError && (
          <div className="card border-red-900/40 bg-red-950/10">
            <p className="text-red-400 text-sm">{fetchError}</p>
          </div>
        )}

        {!fetchError && venues.length === 0 && (
          <div className="card text-center py-12">
            <span className="text-4xl">🏙️</span>
            <p className="text-nite-muted mt-3">No venues listed yet.</p>
          </div>
        )}

        <div className="space-y-3">
          {venues.map((venue) => (
            <Link key={venue.venue_id} href={`/venues/${venue.slug}`} className="block">
              <div className="card hover:border-nite-accent/50 transition-colors">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex-1 min-w-0">
                    <p className="font-semibold">{venue.name}</p>
                    <p className="text-sm text-nite-muted">{venue.city}</p>
                  </div>
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full shrink-0 ${
                      venue.is_active
                        ? 'bg-green-900/40 text-green-400'
                        : 'bg-nite-border text-nite-muted'
                    }`}
                  >
                    {venue.is_active ? 'Open tonight' : 'Closed'}
                  </span>
                </div>
              </div>
            </Link>
          ))}
        </div>

        <Link href="/" className="block text-center text-sm text-nite-muted hover:text-nite-text">
          ← Home
        </Link>
      </main>
    </div>
  )
}
