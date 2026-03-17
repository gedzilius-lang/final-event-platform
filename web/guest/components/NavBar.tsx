// Top navigation bar — server component.
import Link from 'next/link'
import LogoutButton from './LogoutButton'
import { ncToCHF } from '@/lib/api'

interface Props {
  displayName: string
  balance: number | null
}

export default function NavBar({ displayName, balance }: Props) {
  const balanceLabel =
    balance !== null
      ? `${balance.toLocaleString()} NC`
      : '—'

  return (
    <nav className="sticky top-0 z-10 bg-nite-bg/95 backdrop-blur border-b border-nite-border">
      <div className="max-w-lg mx-auto px-4 h-14 flex items-center justify-between gap-3">
        {/* Logo */}
        <Link href="/" className="font-black text-lg tracking-tight shrink-0">
          🌃 <span className="text-nite-accent">NiteOS</span>
        </Link>

        {/* Right: balance + user */}
        <div className="flex items-center gap-2">
          <Link
            href="/wallet"
            className="text-xs bg-nite-surface border border-nite-border rounded-full px-3 py-1 font-semibold text-nite-accent hover:border-nite-accent/50 transition-colors whitespace-nowrap"
          >
            {balanceLabel}
          </Link>
          <LogoutButton displayName={displayName} />
        </div>
      </div>

      {/* Mobile bottom nav */}
      <div className="flex border-t border-nite-border">
        {[
          { href: '/', label: '🏠', text: 'Home' },
          { href: '/events', label: '🏙️', text: 'Events' },
          { href: '/tickets', label: '🎟️', text: 'Tickets' },
          { href: '/wallet', label: '💰', text: 'Wallet' },
        ].map(({ href, label, text }) => (
          <Link
            key={href}
            href={href}
            className="flex-1 flex flex-col items-center py-2 text-xs text-nite-muted hover:text-nite-text transition-colors"
          >
            <span className="text-base leading-tight">{label}</span>
            <span className="leading-tight">{text}</span>
          </Link>
        ))}
      </div>
    </nav>
  )
}
