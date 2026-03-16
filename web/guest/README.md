# web/guest — Guest Web App

Domain: `os.peoplewelike.club`
Framework: Next.js 14, App Router, TypeScript
Port: :3000

Two modes: Global Mode (outside venue) and Venue Mode (after NiteTap check-in).
BFF pattern: all API calls go through Next.js API routes which manage httpOnly cookies.
No direct API calls from browser JavaScript to backend services.

See PRODUCT_BLUEPRINT.md §Product Surfaces for full feature specification.
See FINAL_REPO_STRUCTURE.md for directory layout.

This is a stub. Implementation begins in Phase M4.
