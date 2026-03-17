// Wallet page — shows balance, top-up options.
import { redirect } from 'next/navigation'
import Link from 'next/link'
import { getSession } from '@/lib/session'
import { getWalletBalance, ncToCHF, ApiError } from '@/lib/api'
import NavBar from '@/components/NavBar'
import TopUpButton from './TopUpButton'

// Top-up amounts in CHF. Each gives that many * 100 NC.
const TOPUP_AMOUNTS_CHF = [10, 20, 50, 100]

interface Props {
  searchParams: { success?: string; payment_intent?: string; redirect_status?: string }
}

export default async function WalletPage({ searchParams }: Props) {
  const session = await getSession()
  if (!session.userId) redirect('/login')

  let balance: number | null = null
  let fetchError: string | null = null

  try {
    const w = await getWalletBalance(session.userId, session.accessToken)
    balance = w.balance_nc
  } catch (err) {
    fetchError = err instanceof ApiError ? 'Failed to load balance' : 'Failed to load balance'
  }

  const topupSuccess =
    searchParams.success === '1' ||
    searchParams.redirect_status === 'succeeded'

  return (
    <div className="flex flex-col min-h-screen">
      <NavBar displayName={session.displayName} balance={balance} />

      <main className="flex-1 max-w-lg mx-auto w-full px-4 py-6 space-y-6">
        <h1 className="text-2xl font-bold">Wallet</h1>

        {/* Topup success banner */}
        {topupSuccess && (
          <div className="card border-green-500/40 bg-green-950/20">
            <p className="text-green-400 font-semibold">✓ Top-up successful!</p>
            <p className="text-sm text-nite-muted mt-1">
              Your NiteCoins have been added. Balance may take a moment to update.
            </p>
          </div>
        )}

        {/* Balance card */}
        <div className="card text-center py-8">
          {fetchError ? (
            <p className="text-red-400 text-sm">{fetchError}</p>
          ) : (
            <>
              <p className="text-nite-muted text-sm mb-1">NiteCoin Balance</p>
              <p className="text-5xl font-black text-nite-accent">
                {balance !== null ? balance.toLocaleString() : '—'}
              </p>
              <p className="text-nite-muted text-sm mt-1">NC</p>
              {balance !== null && (
                <p className="text-nite-muted text-xs mt-1">
                  ≈ CHF {ncToCHF(balance)}
                </p>
              )}
            </>
          )}
        </div>

        {/* Top-up grid */}
        <section>
          <h2 className="text-lg font-semibold mb-1">Top up</h2>
          <p className="text-sm text-nite-muted mb-4">
            Add NiteCoins via card payment. 1 CHF = 1 NC.
          </p>
          <div className="grid grid-cols-2 gap-3">
            {TOPUP_AMOUNTS_CHF.map((amount) => (
              <TopUpButton key={amount} amountChf={amount} />
            ))}
          </div>
        </section>

        {/* How it works */}
        <section className="card space-y-2 text-sm">
          <h3 className="font-semibold">How NiteCoins work</h3>
          <ul className="space-y-1.5 text-nite-muted">
            <li>• 1 NC = 1 CHF — fixed peg, no exchange risk</li>
            <li>• Pay at any NiteKiosk by tapping your NiteTap wristband</li>
            <li>• Balance never expires while your account is active</li>
            <li>• Unspent NC refundable on request (Swiss MPV regulation)</li>
          </ul>
        </section>

        <Link href="/" className="block text-center text-sm text-nite-muted hover:text-nite-text">
          ← Home
        </Link>
      </main>
    </div>
  )
}
