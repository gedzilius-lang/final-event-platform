# M0_BLOCKERS.md

Remaining blockers after all autonomous local work is complete.

Last updated: 2026-03-15

---

## Status Summary

All code, configuration, documentation, and scaffolding work has been completed and is ready. The only remaining blockers are external environment prerequisites not available on the current machine. No further code changes are needed to unblock these.

---

## Blocker 1: Go not installed

**Blocks:** M0 exit criterion 2 — `go work` compiles all service stubs

**Symptom:** `go: command not found`

**Impact:** Cannot run `go build`, `go test`, `go work sync`, or `go vet` locally.

**Resolution steps (run in this exact order):**

```bash
# 1. Install Go 1.22+ from https://go.dev/dl/
#    On Windows: download and run the .msi installer
#    On macOS:   brew install go  OR  download .pkg installer
#    On Linux:   sudo apt install golang-go  OR  download tarball

# 2. Verify
go version   # must print go1.22.x or later

# 3. From the repo root, sync workspace (resolves jwt/v5 checksums)
cd /path/to/final-event-platform
go work sync

# 4. Build all services
go build ./services/... ./edge/... ./pkg/...
# Expected: exit 0, no output

# 5. Vet all services
go vet ./services/... ./edge/... ./pkg/...
# Expected: exit 0, no output

# 6. Test all services (stub tests — all pass trivially)
go test ./services/... ./edge/... ./pkg/...
# Expected: all PASS

# 7. Commit the updated go.work.sum
git add go.work.sum
git commit -m "chore: populate go.work.sum after go work sync"
```

**Expected outcome:** M0 exit criterion 2 passes.

---

## Blocker 2: Docker not installed

**Blocks:** M0 exit criterion 3 — `docker compose up` starts Postgres + Redis

**Symptom:** `docker: command not found`

**Impact:** Cannot start the local dev stack. Migrations cannot be run against a live database.

**Resolution steps:**

```bash
# 1. Install Docker Desktop from https://www.docker.com/products/docker-desktop/
#    Accept defaults. Start Docker Desktop.

# 2. Verify
docker --version         # must print Docker version 24.x or later
docker compose version   # must print Docker Compose version 2.x

# 3. Start the dev stack
cd /path/to/final-event-platform
make dev-up
# Expected output:
#   "Waiting for Postgres..."
#   "Dev stack ready. Postgres: localhost:5432  Redis: localhost:6379"

# 4. Verify containers are healthy
docker compose -f infra/docker-compose.dev.yml ps
# Expected: postgres (healthy), redis (healthy)

# 5. Run migrations
make migrate
# Expected: "Migrations complete."

# 6. Seed test data
make seed
```

**Expected outcome:** M0 exit criterion 3 passes.

---

## Blocker 3: No GitHub remote configured

**Blocks:** M0 exit criterion 4 — CI pipeline runs and passes

**Impact:** `.github/workflows/ci.yml` exists and is correct, but CI only runs on GitHub Actions. Without a remote, the workflow cannot be triggered.

**Resolution steps:**

```bash
# 1. Create a new repository on GitHub (or use existing org)
#    Recommended name: niteos  (or: final-event-platform)
#    Visibility: private

# 2. Add the remote
cd /path/to/final-event-platform
git remote add origin git@github.com:<your-org>/niteos.git

# 3. Push
git push -u origin main

# 4. Open GitHub → Actions tab
#    The CI workflow should trigger automatically on push to main
#    Expected: all jobs pass (go build, vet, test, lint, migrations)
```

**Expected outcome:** M0 exit criterion 4 passes (CI green).

---

---

## Verification Step: Confirm PEM keys are not tracked in git

**Not a blocker — but must be verified before first push.**

The `.gitignore` rules (`infra/secrets/*` and `*.pem`) should prevent the dev PEM keys from being committed. Verify this before pushing:

```bash
cd /path/to/final-event-platform

# Confirm PEM files are NOT tracked
git ls-files infra/secrets/
# Expected output: (nothing — or only .gitkeep if it has been added)

# If PEM files appear in the output, remove them from tracking:
git rm --cached infra/secrets/jwt_private_key.pem
git rm --cached infra/secrets/jwt_public_key.pem

# Add the directory placeholder so infra/secrets/ exists on fresh clone
git add infra/secrets/.gitkeep
git status
# Expected: infra/secrets/.gitkeep listed as new file; PEM files NOT listed
```

---

## Not a Blocker: TWINT credentials

TWINT credentials are not needed for M0 or M1. They are needed for M3.2 (payments service implementation). A Stripe mock and Stripe test mode are used first. TWINT is a provider interface swap when Swiss business account credentials arrive.

## Not a Blocker: Production secrets

Production Postgres password, Redis password, Stripe keys, Cloudflare API token are not needed until M5 (Infrastructure Consolidation). The dev stack uses the placeholder credentials defined in `infra/docker-compose.dev.yml` and `.env.example`.

---

## After all blockers are cleared

Once Go, Docker, and GitHub remote are in place, run this validation sequence:

```bash
# Full local validation
go work sync
go build ./services/... ./edge/... ./pkg/...
go vet ./services/... ./edge/... ./pkg/...
go test ./services/... ./edge/... ./pkg/...
make dev-up
make migrate

# Then push to GitHub and confirm CI green
git push origin main
# Open GitHub Actions and confirm all jobs pass

# Then update PHASE_1_STATUS.md exit criteria to PASS and proceed to M1
```
