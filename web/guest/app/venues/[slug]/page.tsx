// Venue detail page — shows venue info and catalog items (read-only, for browsing).
import { redirect } from 'next/navigation'
import Link from 'next/link'
import { getSession } from '@/lib/session'
import { getVenues, getWalletBalance, backendFetch, VenueSummary, ncToCHF } from '@/lib/api'
import NavBar from '@/components/NavBar'

interface CatalogItem {
  item_id: string
  name: string
  category: string
  price_nc: number
  icon?: string
  is_active: boolean
  display_order: number
  happy_hour_price_nc?: number
}

interface Props {
  params: { slug: string }
}

export default async function VenuePage({ params }: Props) {
  const session = await getSession()
  if (!session.userId) redirect('/login')

  const [venuesResult, balanceResult] = await Promise.allSettled([
    getVenues(session.accessToken),
    getWalletBalance(session.userId, session.accessToken),
  ])

  const venues =
    venuesResult.status === 'fulfilled' ? venuesResult.value.venues : []
  const venue = venues.find((v) => v.slug === params.slug)
  const balance =
    balanceResult.status === 'fulfilled' ? balanceResult.value.balance_nc : null

  if (!venue) {
    return (
      <div className="flex flex-col min-h-screen">
        <NavBar displayName={session.displayName} balance={balance} />
        <main className="flex-1 max-w-lg mx-auto w-full px-4 py-6">
          <p className="text-nite-muted">Venue not found.</p>
          <Link href="/" className="text-nite-accent text-sm mt-4 block">← Back</Link>
        </main>
      </div>
    )
  }

  // Fetch catalog items — non-fatal
  let items: CatalogItem[] = []
  try {
    const r = await backendFetch<{ items: CatalogItem[] }>(
      `/catalog/venues/${venue.venue_id}/items`,
      session.accessToken,
    )
    items = r.items
      .filter((i) => i.is_active)
      .sort((a, b) => a.display_order - b.display_order)
  } catch {
    // Non-fatal — show venue without catalog
  }

  const categories = Array.from(new Set(items.map((i) => i.category)))

  return (
    <div className="flex flex-col min-h-screen">
      <NavBar displayName={session.displayName} balance={balance} />

      <main className="flex-1 max-w-lg mx-auto w-full px-4 py-6 space-y-6">
        {/* Venue header */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <h1 className="text-2xl font-bold">{venue.name}</h1>
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${
                venue.is_active
                  ? 'bg-green-900/40 text-green-400'
                  : 'bg-nite-border text-nite-muted'
              }`}
            >
              {venue.is_active ? 'Open' : 'Closed'}
            </span>
          </div>
          <p className="text-nite-muted text-sm">{venue.city}</p>
        </div>

        {/* NiteTap hint */}
        {venue.is_active && (
          <div className="card border-nite-accent/30 bg-amber-950/10">
            <p className="text-sm font-semibold text-nite-accent mb-1">How to enter</p>
            <p className="text-sm text-nite-muted">
              Tap your NiteTap wristband on the NiteKiosk at the entrance.
              Your session opens automatically.
            </p>
            <p className="text-xs text-nite-muted mt-2">
              No NiteTap? Pick one up at the door.
            </p>
          </div>
        )}

        {/* Menu */}
        {items.length > 0 && (
          <section>
            <h2 className="text-lg font-semibold mb-3">Menu</h2>
            <div className="space-y-4">
              {categories.map((cat) => (
                <div key={cat}>
                  <h3 className="text-xs font-semibold text-nite-muted uppercase tracking-wider mb-2 capitalize">
                    {cat}
                  </h3>
                  <div className="space-y-2">
                    {items
                      .filter((i) => i.category === cat)
                      .map((item) => (
                        <div key={item.item_id} className="card flex items-center justify-between gap-3">
                          <div className="flex items-center gap-3">
                            {item.icon && <span className="text-xl">{item.icon}</span>}
                            <span className="font-medium text-sm">{item.name}</span>
                          </div>
                          <div className="text-right shrink-0">
                            <p className="text-nite-accent font-semibold text-sm">
                              {item.price_nc.toLocaleString()} NC
                            </p>
                            <p className="text-nite-muted text-xs">
                              CHF {ncToCHF(item.price_nc)}
                            </p>
                          </div>
                        </div>
                      ))}
                  </div>
                </div>
              ))}
            </div>
            <p className="text-xs text-nite-muted mt-3">
              Order at any NiteKiosk terminal. Tap your NiteTap to confirm.
            </p>
          </section>
        )}

        {items.length === 0 && venue.is_active && (
          <p className="text-sm text-nite-muted">Menu not available yet.</p>
        )}

        <Link href="/" className="block text-center text-sm text-nite-muted hover:text-nite-text">
          ← Back
        </Link>
      </main>
    </div>
  )
}
