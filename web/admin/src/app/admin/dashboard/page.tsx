import { getSession } from '@/lib/session'
import { backendFetch, type VenueRevenue } from '@/lib/api'

function StatCard({ label, value, sub }: { label: string; value: string | number; sub?: string }) {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-5">
      <p className="text-xs text-gray-400 uppercase tracking-wider">{label}</p>
      <p className="mt-1 text-2xl font-semibold text-white">{value}</p>
      {sub && <p className="mt-1 text-xs text-gray-500">{sub}</p>}
    </div>
  )
}

export default async function DashboardPage() {
  const session = await getSession()

  let revenue: VenueRevenue | null = null
  let revenueError = ''

  const today = new Date()
  const firstOfMonth = new Date(today.getFullYear(), today.getMonth(), 1)
    .toISOString().split('T')[0]
  const todayStr = today.toISOString().split('T')[0]

  if (session.venueId) {
    try {
      revenue = await backendFetch<VenueRevenue>(
        `/reporting/venues/${session.venueId}/revenue?from=${firstOfMonth}&to=${todayStr}`,
        session.accessToken,
      )
    } catch (e: unknown) {
      revenueError = (e as Error).message
    }
  }

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold text-white mb-1">Dashboard</h1>
      <p className="text-sm text-gray-400 mb-8">Month-to-date · {todayStr}</p>

      {revenueError && (
        <div className="mb-6 rounded-lg bg-red-950/40 border border-red-800 px-4 py-3 text-sm text-red-300">
          Revenue data unavailable: {revenueError}
        </div>
      )}

      {revenue ? (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
          <StatCard label="Orders (CHF)" value={(revenue.total_orders_nc / 100).toFixed(2)} sub="NC → CHF at 1:1" />
          <StatCard label="Orders" value={revenue.order_count} />
          <StatCard label="Sessions" value={revenue.session_count} />
          <StatCard label="Top-ups (CHF)" value={(revenue.total_topups_nc / 100).toFixed(2)} />
        </div>
      ) : session.venueId ? (
        <p className="text-sm text-gray-500 mb-8">Loading venue stats…</p>
      ) : (
        <div className="mb-8 rounded-lg bg-gray-900 border border-gray-800 px-6 py-8 text-center">
          <p className="text-gray-400 text-sm">
            {session.role === 'nitecore'
              ? 'Nitecore view — venue-specific stats require selecting a venue.'
              : 'No venue associated with your account.'}
          </p>
        </div>
      )}

      <div className="rounded-xl bg-gray-900 border border-gray-800 px-6 py-5">
        <h2 className="text-sm font-medium text-gray-300 mb-3">Quick actions</h2>
        <div className="flex flex-wrap gap-3">
          <a href="/admin/catalog/items/new"
            className="rounded-lg bg-brand-500 hover:bg-brand-600 px-4 py-2 text-sm font-medium text-white transition-colors">
            + Add catalog item
          </a>
          <a href="/admin/devices"
            className="rounded-lg bg-gray-800 hover:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-200 transition-colors">
            Manage devices
          </a>
          <a href="/admin/sessions"
            className="rounded-lg bg-gray-800 hover:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-200 transition-colors">
            Live sessions
          </a>
          <a href="/admin/reports"
            className="rounded-lg bg-gray-800 hover:bg-gray-700 px-4 py-2 text-sm font-medium text-gray-200 transition-colors">
            Revenue report
          </a>
        </div>
      </div>
    </div>
  )
}
