import { getSession } from '@/lib/session'
import { backendFetch, type VenueRevenue } from '@/lib/api'

export default async function ReportsPage({
  searchParams,
}: {
  searchParams: { from?: string; to?: string }
}) {
  const session = await getSession()
  const venueId = session.venueId ?? ''

  const today = new Date()
  const defaultFrom = new Date(today.getFullYear(), today.getMonth(), 1)
    .toISOString()
    .split('T')[0]
  const defaultTo = today.toISOString().split('T')[0]

  const from = searchParams.from ?? defaultFrom
  const to   = searchParams.to   ?? defaultTo

  let revenue: VenueRevenue | null = null
  let err = ''

  if (venueId) {
    try {
      revenue = await backendFetch<VenueRevenue>(
        `/reporting/venues/${venueId}/revenue?from=${from}&to=${to}`,
        session.accessToken,
      )
    } catch (e: unknown) {
      err = (e as Error).message
    }
  }

  const ncToCHF = (nc: number) => (nc / 100).toFixed(2)

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold text-white mb-2">Revenue Report</h1>

      <form method="GET" className="flex items-end gap-4 mb-8">
        <div>
          <label className="block text-xs text-gray-400 mb-1">From</label>
          <input
            type="date"
            name="from"
            defaultValue={from}
            className="rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 text-sm text-white
                       focus:outline-none focus:ring-2 focus:ring-brand-500"
          />
        </div>
        <div>
          <label className="block text-xs text-gray-400 mb-1">To</label>
          <input
            type="date"
            name="to"
            defaultValue={to}
            className="rounded-lg bg-gray-800 border border-gray-700 px-3 py-2 text-sm text-white
                       focus:outline-none focus:ring-2 focus:ring-brand-500"
          />
        </div>
        <button
          type="submit"
          className="rounded-lg bg-gray-700 hover:bg-gray-600 px-4 py-2 text-sm text-white"
        >
          Apply
        </button>
      </form>

      {!venueId && (
        <p className="text-sm text-gray-400">No venue associated with this account.</p>
      )}

      {err && (
        <div className="mb-4 rounded-lg bg-red-950/40 border border-red-800 px-4 py-3 text-sm text-red-300">
          {err}
        </div>
      )}

      {revenue && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            {[
              { label: 'Orders (CHF)', value: ncToCHF(revenue.total_orders_nc) },
              { label: 'Orders count', value: revenue.order_count },
              { label: 'Top-ups (CHF)', value: ncToCHF(revenue.total_topups_nc) },
              { label: 'Sessions', value: revenue.session_count },
            ].map(s => (
              <div key={s.label} className="bg-gray-900 border border-gray-800 rounded-xl p-5">
                <p className="text-xs text-gray-400 uppercase tracking-wider">{s.label}</p>
                <p className="mt-1 text-2xl font-semibold text-white">{s.value}</p>
              </div>
            ))}
          </div>

          <div className="bg-gray-900 border border-gray-800 rounded-xl p-5">
            <h2 className="text-sm font-medium text-gray-300 mb-3">Settlement estimate</h2>
            <p className="text-xs text-gray-400 mb-4">
              95% of order volume goes to venue. 5% is NiteOS platform fee.
            </p>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-xs text-gray-500">Venue receives (CHF)</p>
                <p className="text-lg font-semibold text-green-400">
                  {ncToCHF(Math.floor(revenue.total_orders_nc * 0.95))}
                </p>
              </div>
              <div>
                <p className="text-xs text-gray-500">Platform fee (CHF)</p>
                <p className="text-lg font-semibold text-gray-300">
                  {ncToCHF(Math.floor(revenue.total_orders_nc * 0.05))}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
