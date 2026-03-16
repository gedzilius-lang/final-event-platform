import { redirect } from 'next/navigation'

// Redirect legacy /dashboard → /admin/dashboard (canonical URL)
export default function DashboardRedirect() {
  redirect('/admin/dashboard')
}
