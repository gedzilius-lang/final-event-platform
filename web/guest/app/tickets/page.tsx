// My Tickets page
import { redirect } from 'next/navigation'
import Link from 'next/link'
import { getSession } from '@/lib/session'
import { getMyTickets, getWalletBalance, Ticket, ApiError } from '@/lib/api'
import NavBar from '@/components/NavBar'

export default async function TicketsPage() {
  const session = await getSession()
  if (!session.userId) redirect('/login')

  const [ticketsResult, balanceResult] = await Promise.allSettled([
    getMyTickets(session.userId, session.accessToken),
    getWalletBalance(session.userId, session.accessToken),
  ])

  const tickets =
    ticketsResult.status === 'fulfilled' ? ticketsResult.value.tickets : []
  const fetchError =
    ticketsResult.status === 'rejected'
      ? 'Failed to load tickets'
      : null
  const balance =
    balanceResult.status === 'fulfilled' ? balanceResult.value.balance_nc : null

  const active = tickets.filter((t) => t.status === 'active')
  const past = tickets.filter((t) => t.status !== 'active')

  return (
    <div className="flex flex-col min-h-screen">
      <NavBar displayName={session.displayName} balance={balance} />

      <main className="flex-1 max-w-lg mx-auto w-full px-4 py-6 space-y-6">
        <h1 className="text-2xl font-bold">My Tickets</h1>

        {fetchError && (
          <div className="card border-red-900/40 bg-red-950/10">
            <p className="text-red-400 text-sm">{fetchError}</p>
          </div>
        )}

        {!fetchError && tickets.length === 0 && (
          <div className="card text-center py-12">
            <span className="text-4xl">🎟️</span>
            <p className="text-nite-muted mt-3 mb-4">No tickets yet.</p>
            <Link href="/events" className="btn-primary text-sm">
              Browse venues
            </Link>
          </div>
        )}

        {active.length > 0 && (
          <section>
            <h2 className="text-xs font-semibold text-nite-muted uppercase tracking-wider mb-3">
              Active tickets
            </h2>
            <div className="space-y-3">
              {active.map((t) => <TicketCard key={t.ticket_id} ticket={t} />)}
            </div>
          </section>
        )}

        {past.length > 0 && (
          <section>
            <h2 className="text-xs font-semibold text-nite-muted uppercase tracking-wider mb-3">
              Past tickets
            </h2>
            <div className="space-y-3 opacity-60">
              {past.map((t) => <TicketCard key={t.ticket_id} ticket={t} />)}
            </div>
          </section>
        )}

        <Link href="/" className="block text-center text-sm text-nite-muted hover:text-nite-text">
          ← Home
        </Link>
      </main>
    </div>
  )
}

const statusColor: Record<string, string> = {
  active: 'bg-green-900/40 text-green-400',
  used: 'bg-nite-border text-nite-muted',
  expired: 'bg-nite-border text-nite-muted',
  cancelled: 'bg-red-900/40 text-red-400',
}

function TicketCard({ ticket }: { ticket: Ticket }) {
  const validDate = new Date(ticket.valid_from).toLocaleDateString('de-CH', {
    weekday: 'short',
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })

  return (
    <div className="card">
      <div className="flex items-start justify-between gap-3">
        <div className="flex-1 min-w-0">
          <p className="font-semibold truncate">{ticket.event_name}</p>
          <p className="text-sm text-nite-muted mt-0.5">{validDate}</p>
        </div>
        <span
          className={`text-xs px-2 py-0.5 rounded-full shrink-0 capitalize ${statusColor[ticket.status] ?? 'bg-nite-border text-nite-muted'}`}
        >
          {ticket.status}
        </span>
      </div>

      {ticket.status === 'active' && (
        <div className="mt-4 pt-4 border-t border-nite-border">
          <p className="text-xs text-nite-muted mb-2">Show at entrance</p>
          {/* QR code as large monospace text — scannable by NiteTerminal camera */}
          <div className="bg-white rounded-lg p-4 text-center">
            <p className="text-black font-mono text-xs break-all leading-relaxed">
              {ticket.qr_payload}
            </p>
          </div>
          <p className="text-xs text-nite-muted mt-2">
            Present this screen to the door terminal. Keep screen brightness high.
          </p>
        </div>
      )}
    </div>
  )
}
