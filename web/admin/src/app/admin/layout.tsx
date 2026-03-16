import { redirect } from 'next/navigation'
import { getSession } from '@/lib/session'
import NavSidebar from '@/components/NavSidebar'

export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const session = await getSession()
  if (!session.userId) redirect('/login')

  return (
    <div className="flex min-h-screen">
      <NavSidebar
        role={session.role}
        displayName={session.displayName}
        venueId={session.venueId}
      />
      <main className="flex-1 overflow-auto">
        {children}
      </main>
    </div>
  )
}
