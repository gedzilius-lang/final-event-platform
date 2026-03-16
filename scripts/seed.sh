#!/usr/bin/env bash
# Seed the local database with development test data.
# For development use only. Never run against production.
# Requires: migrations to have already run (make migrate).
set -euo pipefail

DB="${DATABASE_URL:-postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable}"

echo "==> Seeding dev database..."
psql "$DB" -f migrations/seed_dev.sql
echo ""
echo "Seed complete. Test credentials (all passwords: devpassword):"
echo "  nitecore@niteos.dev        — role: nitecore"
echo "  admin@venue-alpha.dev      — role: venue_admin"
echo "  bartender@venue-alpha.dev  — role: bartender"
echo "  door@venue-alpha.dev       — role: door_staff"
echo "  guest@example.dev          — role: guest (500 NC pre-loaded)"
echo ""
echo "Test venue: 'Venue Alpha' (slug: venue-alpha)"
echo "Staff PIN:  1234"
