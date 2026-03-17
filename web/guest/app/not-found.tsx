import Link from 'next/link'

export default function NotFound() {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center px-4 text-center">
      <span className="text-5xl mb-4">🌌</span>
      <h1 className="text-2xl font-bold mb-2">Page not found</h1>
      <p className="text-nite-muted text-sm mb-6">
        This page doesn&apos;t exist or has moved.
      </p>
      <Link href="/" className="btn-primary text-sm">
        Back to home
      </Link>
    </div>
  )
}
