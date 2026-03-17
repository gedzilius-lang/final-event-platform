// Security surface — guest lookup, flag, incident notes.
import { redirect } from 'next/navigation'
import { requireSession } from '@/lib/session'
import SecurityPanel from './SecurityPanel'

export default async function SecurityPage() {
  const session = await requireSession()
  if (!['security', 'door_staff', 'venue_admin', 'nitecore'].includes(session.role)) {
    redirect('/staff')
  }

  return (
    <div className="space-y-5">
      <h1 className="text-sm font-semibold uppercase tracking-wider text-nite-muted">
        Guest Lookup
      </h1>
      <SecurityPanel />
    </div>
  )
}
