import { redirect } from 'next/navigation'

// The (admin) route group is no longer active. All admin pages live under /admin/*.
// This layout redirects any lingering (admin) group routes to /admin.
export default function LegacyAdminLayout({ children }: { children: React.ReactNode }) {
  redirect('/admin')
  return <>{children}</>
}
