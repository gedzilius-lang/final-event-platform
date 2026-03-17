import Link from 'next/link'
import { ncToCHF } from '@/lib/api'

export default function WalletBadge({ balance }: { balance: number | null }) {
  return (
    <div className="card flex items-center justify-between">
      <div>
        <p className="text-xs text-nite-muted uppercase tracking-wider">Wallet</p>
        <p className="text-3xl font-black text-nite-accent mt-0.5">
          {balance !== null ? balance.toLocaleString() : '—'}{' '}
          <span className="text-base font-normal text-nite-muted">NC</span>
        </p>
        {balance !== null && (
          <p className="text-xs text-nite-muted">≈ CHF {ncToCHF(balance)}</p>
        )}
      </div>
      <Link href="/wallet" className="btn-primary text-sm">
        Top up
      </Link>
    </div>
  )
}
