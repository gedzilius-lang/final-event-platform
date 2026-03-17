// Bar staff POS — fast order entry, NiteTap charge.
import { redirect } from 'next/navigation'
import { requireSession } from '@/lib/session'
import { backendFetch, ApiError } from '@/lib/api'
import BarPOS from './BarPOS'

interface CatalogItem {
  item_id: string
  name: string
  category: string
  price_nc: number
  icon?: string
  is_active: boolean
  display_order: number
}

export default async function BarPage() {
  const session = await requireSession()
  if (!['bartender', 'venue_admin', 'nitecore'].includes(session.role)) redirect('/staff')
  if (!session.venueId) {
    return (
      <div className="card text-center py-10">
        <p className="text-nite-muted text-sm">
          No venue assigned to your account. Ask your manager to assign a venue.
        </p>
      </div>
    )
  }

  let items: CatalogItem[] = []
  try {
    const r = await backendFetch<{ items: CatalogItem[] }>(
      `/catalog/venues/${session.venueId}/items`,
      session.accessToken,
    )
    items = r.items.filter((i) => i.is_active).sort((a, b) => a.display_order - b.display_order)
  } catch (e) {
    if (!(e instanceof ApiError)) throw e
  }

  return <BarPOS items={items} venueId={session.venueId} />
}
