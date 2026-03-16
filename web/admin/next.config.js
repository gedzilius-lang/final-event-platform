/** @type {import('next').NextConfig} */
const nextConfig = {
  // All API calls are BFF — no direct client-to-backend calls
  // Backend service URLs are server-only env vars (never prefixed with NEXT_PUBLIC_)
  reactStrictMode: true,
  // Standalone output: self-contained server bundle for Docker deployment.
  output: 'standalone',
}

module.exports = nextConfig
