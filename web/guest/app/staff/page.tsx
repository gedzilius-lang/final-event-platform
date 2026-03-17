// Staff index — redirects each role to its primary surface.
import { redirect } from 'next/navigation'
import { requireSession } from '@/lib/session'

export default async function StaffIndex() {
  const session = await requireSession()
  switch (session.role) {
    case 'door_staff': redirect('/staff/door')
    case 'bartender': redirect('/staff/bar')
    case 'security': redirect('/staff/security')
    case 'venue_admin':
    case 'nitecore': redirect('/staff/manager')
    default: redirect('/')
  }
}
