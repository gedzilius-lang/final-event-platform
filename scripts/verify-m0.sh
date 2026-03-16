#!/usr/bin/env bash
# scripts/verify-m0.sh
# Runs the full M0 validation sequence.
# Every exit criterion from MIGRATION_PLAN.md M0 is tested here.
#
# Prerequisites: Go 1.22+, Docker, Make must all be installed.
# Usage: bash scripts/verify-m0.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

log()  { echo "[verify-m0] $*"; }
pass() { echo "[verify-m0] PASS: $*"; }
fail() { echo "[verify-m0] FAIL: $*" >&2; exit 1; }

echo ""
echo "NiteOS M0 Verification"
echo "======================"
echo ""

# ── Criterion 1: Monorepo structure ───────────────────────────────────────────
log "Checking directory structure..."

required_dirs=(
  "services/gateway" "services/auth" "services/profiles" "services/ledger"
  "services/wallet" "services/payments" "services/ticketing" "services/orders"
  "services/catalog" "services/devices" "services/sessions" "services/reporting"
  "services/sync" "edge" "pkg/jwtutil" "pkg/middleware" "pkg/httputil"
  "pkg/idempotency" "web/guest" "web/admin" "android/nitekiosk"
  "android/niteterminal" "android/mastertablet" "android/shared"
  "infra" "migrations" ".github/workflows"
)

for d in "${required_dirs[@]}"; do
  [[ -d "$d" ]] || fail "Directory missing: $d"
done
pass "Directory structure complete"

# ── Criterion 2: go work compiles ─────────────────────────────────────────────
log "Checking Go workspace compilation..."

command -v go &>/dev/null || fail "Go is not installed. Run: bash scripts/install-prereqs.sh"

go work sync
log "go work sync completed"

go build ./services/... ./edge/... ./pkg/...
pass "go build: all service stubs compile"

go vet ./services/... ./edge/... ./pkg/...
pass "go vet: no issues"

go test ./services/... ./edge/... ./pkg/...
pass "go test: all tests pass"

# ── Criterion 3: docker compose starts Postgres + Redis ───────────────────────
log "Checking Docker..."
command -v docker &>/dev/null || fail "Docker is not installed. Run: bash scripts/install-prereqs.sh"

docker compose -f infra/docker-compose.dev.yml config --quiet
pass "docker compose config: valid"

log "Starting dev stack (this may take up to 30 seconds for first pull)..."
docker compose -f infra/docker-compose.dev.yml up -d --wait --wait-timeout 60
pass "docker compose up: stack is up"

log "Verifying Postgres health..."
docker compose -f infra/docker-compose.dev.yml exec -T postgres \
  pg_isready -U niteos -d niteos -q
pass "Postgres: healthy"

log "Verifying Redis health..."
docker compose -f infra/docker-compose.dev.yml exec -T redis \
  redis-cli -a devpassword ping | grep -q "PONG"
pass "Redis: healthy"

# ── Criterion 4: migrations run cleanly ───────────────────────────────────────
log "Running migrations..."
bash scripts/migrate.sh
pass "migrations: all passed"

# ── Criterion 5: RS256 key pair ───────────────────────────────────────────────
log "Checking RS256 key pair..."
[[ -f "infra/secrets/jwt_private_key.pem" ]] || {
  log "Key pair not found. Generating..."
  mkdir -p infra/secrets
  openssl genrsa -out infra/secrets/jwt_private_key.pem 2048
  openssl rsa -in infra/secrets/jwt_private_key.pem -pubout \
    -out infra/secrets/jwt_public_key.pem
  log "Key pair generated."
}
[[ -f "infra/secrets/jwt_public_key.pem" ]] || fail "Public key missing"
pass "RS256 key pair: present"

# ── git tracking check ────────────────────────────────────────────────────────
log "Checking PEM files are not tracked in git..."
tracked=$(git ls-files infra/secrets/ 2>/dev/null | grep -v '\.gitkeep' || true)
if [[ -n "$tracked" ]]; then
  fail "PEM files are tracked in git! Run: git rm --cached $tracked"
fi
pass "PEM files: not tracked in git"

# ── go.work.sum present ───────────────────────────────────────────────────────
[[ -f "go.work.sum" ]] || fail "go.work.sum is missing — run: go work sync"
pass "go.work.sum: present"

echo ""
echo "M0 VERIFICATION COMPLETE"
echo "All 5 exit criteria pass."
echo ""
echo "Next steps:"
echo "  1. Add GitHub remote: git remote add origin <url>"
echo "  2. Push: git push -u origin main"
echo "  3. Confirm CI passes in GitHub Actions"
echo "  4. Mark M0 as COMPLETE in PHASE_1_STATUS.md"
echo ""
