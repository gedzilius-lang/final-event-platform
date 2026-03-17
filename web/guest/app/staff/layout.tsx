// Staff layout — server component. Role guard: guests are redirected to /.
// Role enforcement happens here + in each surface via requireSession().
import { redirect } from 'next/navigation'
import Link from 'next/link'
import { requireSession } from '@/lib/session'

const STAFF_ROLES = ['door_staff', 'bartender', 'security', 'venue_admin', 'nitecore']

export default async function StaffLayout({ children }: { children: React.ReactNode }) {
  const session = await requireSession().catch(() => null)
  if (!session) redirect('/login?next=/staff')
  if (!STAFF_ROLES.includes(session.role)) redirect('/')

  const navItems = buildNavItems(session.role)

  return (
    <div className="flex flex-col min-h-screen bg-nite-bg text-nite-text">
      {/* Staff top bar */}
      <header className="sticky top-0 z-10 bg-nite-surface border-b border-nite-border">
        <div className="max-w-lg mx-auto px-4 h-12 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <span className="text-nite-accent font-black text-sm">NiteOS</span>
            <span className="text-nite-border">·</span>
            <span className="text-xs font-semibold text-nite-muted uppercase tracking-wider">
              {roleLabel(session.role)}
            </span>
          </div>
          <Link href="/" className="text-xs text-nite-muted hover:text-nite-text transition-colors">
            ← Guest app
          </Link>
        </div>
      </header>

      {/* Staff nav */}
      {navItems.length > 1 && (
        <nav className="bg-nite-surface border-b border-nite-border">
          <div className="max-w-lg mx-auto px-4 flex gap-1 overflow-x-auto">
            {navItems.map(({ href, label }) => (
              <Link
                key={href}
                href={href}
                className="shrink-0 px-3 py-2.5 text-xs font-semibold text-nite-muted hover:text-nite-text transition-colors border-b-2 border-transparent hover:border-nite-accent"
              >
                {label}
              </Link>
            ))}
          </div>
        </nav>
      )}

      {/* Content */}
      <main className="flex-1 max-w-lg mx-auto w-full px-4 py-5">
        {children}
      </main>
    </div>
  )
}

function roleLabel(role: string): string {
  switch (role) {
    case 'door_staff': return 'Door'
    case 'bartender': return 'Bar'
    case 'security': return 'Security'
    case 'venue_admin': return 'Manager'
    case 'nitecore': return 'Admin'
    default: return role
  }
}

function buildNavItems(role: string): { href: string; label: string }[] {
  if (role === 'nitecore' || role === 'venue_admin') {
    return [
      { href: '/staff/manager', label: 'Overview' },
      { href: '/staff/door', label: 'Door' },
      { href: '/staff/bar', label: 'Bar POS' },
      { href: '/staff/security', label: 'Security' },
    ]
  }
  if (role === 'door_staff') return [{ href: '/staff/door', label: 'Door' }]
  if (role === 'bartender') return [{ href: '/staff/bar', label: 'Bar POS' }]
  if (role === 'security') return [{ href: '/staff/security', label: 'Security' }]
  return []
}
