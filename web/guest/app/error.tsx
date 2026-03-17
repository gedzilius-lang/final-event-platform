'use client'
import Link from 'next/link'
import { useEffect } from 'react'

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    console.error('Page error:', error)
  }, [error])

  return (
    <div className="min-h-screen flex flex-col items-center justify-center px-4 text-center">
      <span className="text-5xl mb-4">⚠️</span>
      <h1 className="text-xl font-bold mb-2">Something went wrong</h1>
      <p className="text-nite-muted text-sm mb-6 max-w-xs">
        An error occurred loading this page. Your session and wallet are unaffected.
      </p>
      <div className="flex gap-3">
        <button onClick={reset} className="btn-primary text-sm">
          Try again
        </button>
        <Link href="/" className="btn-ghost text-sm">
          Home
        </Link>
      </div>
    </div>
  )
}
