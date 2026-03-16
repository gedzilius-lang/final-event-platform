'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import clsx from 'clsx'

const nav = [
  { href: '/admin/dashboard',        label: 'Dashboard',   icon: '⚡' },
  { href: '/admin/catalog/venues',   label: 'Venues',      icon: '🏛️' },
  { href: '/admin/catalog/items',    label: 'Catalog',     icon: '🍺' },
  { href: '/admin/devices',          label: 'Devices',     icon: '📱' },
  { href: '/admin/sessions',         label: 'Sessions',    icon: '🎫' },
  { href: '/admin/reports',          label: 'Reports',     icon: '📊' },
  { href: '/admin/users',            label: 'Users',       icon: '👤' },
]

interface Props {
  role: string
  displayName: string
  venueId?: string
}

export default function NavSidebar({ role, displayName }: Props) {
  const pathname = usePathname()
  const router = useRouter()

  async function logout() {
    await fetch('/api/auth/logout', { method: 'POST' })
    router.push('/login')
    router.refresh()
  }

  return (
    <aside className="w-56 min-h-screen bg-gray-900 border-r border-gray-800 flex flex-col">
      <div className="px-4 py-5 border-b border-gray-800">
        <span className="text-lg font-bold text-white tracking-tight">NiteOS</span>
        <span className="ml-2 text-xs bg-brand-500/20 text-brand-100 px-1.5 py-0.5 rounded">
          {role}
        </span>
      </div>

      <nav className="flex-1 px-2 py-4 space-y-0.5">
        {nav.map(item => (
          <Link
            key={item.href}
            href={item.href}
            className={clsx(
              'flex items-center gap-2.5 px-3 py-2 rounded-md text-sm transition-colors',
              pathname.startsWith(item.href)
                ? 'bg-brand-500/20 text-brand-100 font-medium'
                : 'text-gray-400 hover:text-white hover:bg-gray-800',
            )}
          >
            <span>{item.icon}</span>
            {item.label}
          </Link>
        ))}
      </nav>

      <div className="px-4 py-4 border-t border-gray-800 space-y-1">
        <p className="text-xs text-gray-500 truncate">{displayName}</p>
        <button
          onClick={logout}
          className="text-xs text-gray-400 hover:text-white transition-colors"
        >
          Sign out
        </button>
      </div>
    </aside>
  )
}
