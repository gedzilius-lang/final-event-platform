'use client'
import { useRouter } from 'next/navigation'

export default function LogoutButton({ displayName }: { displayName: string }) {
  const router = useRouter()

  async function handleLogout() {
    await fetch('/api/auth/logout', { method: 'POST' })
    router.push('/')
    router.refresh()
  }

  return (
    <button
      onClick={handleLogout}
      title={`Signed in as ${displayName} — click to sign out`}
      className="text-xs text-nite-muted hover:text-nite-text transition-colors max-w-[80px] truncate"
    >
      {displayName}
    </button>
  )
}
