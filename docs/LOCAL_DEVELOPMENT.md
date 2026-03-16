# LOCAL_DEVELOPMENT.md

Complete guide for setting up and running the NiteOS development environment on a local machine.

---

## Prerequisites

Install these tools before anything else:

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.22+ | https://go.dev/dl/ — download and install; verify: `go version` |
| Docker Desktop | Latest stable | https://www.docker.com/products/docker-desktop/ — verify: `docker --version` |
| openssl | Any modern | Usually pre-installed on macOS/Linux; Windows: Git Bash includes it |
| psql | PostgreSQL 15+ client | Installed with PostgreSQL or via `brew install libpq` (macOS) |
| golang-migrate CLI | v4.17+ | See install instructions below |

### Install golang-migrate

**macOS:**
```bash
brew install golang-migrate
```

**Linux:**
```bash
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
```

**Windows (Git Bash):**
```bash
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.windows-amd64.zip -o migrate.zip
unzip migrate.zip
mv migrate.exe /usr/local/bin/
```

---

## First-Time Setup

Run these steps once after cloning the repository.

### 1. Sync workspace and resolve module checksums

```bash
cd /path/to/final-event-platform
go work sync
```

This resolves external module dependencies (currently: `github.com/golang-jwt/jwt/v5`) and populates `go.work.sum`. Commit the updated `go.work.sum` if it changes.

### 2. Generate the RS256 JWT key pair

```bash
mkdir -p infra/secrets
openssl genrsa -out infra/secrets/jwt_private_key.pem 2048
openssl rsa -in infra/secrets/jwt_private_key.pem -pubout -out infra/secrets/jwt_public_key.pem
```

These files are gitignored (`infra/secrets/*`). Never commit them. They stay on your local machine only. See `docs/SECRET_ROTATION.md` for production key management.

### 3. Copy the example env file

```bash
cp .env.example .env
```

The defaults in `.env.example` match the local dev stack. No changes needed for basic local development.

---

## Starting the Dev Stack

The dev stack consists of Postgres 16 and Redis 7. Go services run directly (not via Docker) during development.

```bash
make dev-up
```

This starts Postgres on `localhost:5432` and Redis on `localhost:6379`, then runs all pending migrations.

**Credentials (dev only):**
- Postgres: `niteos:devpassword` / database: `niteos`
- Redis: password `devpassword`

To verify the stack is healthy:
```bash
docker compose -f infra/docker-compose.dev.yml ps
```

---

## Running a Service Locally

Each Go service runs as a standalone process. Example for the ledger service:

```bash
cd services/ledger
DATABASE_URL="postgres://niteos:devpassword@localhost:5432/niteos?sslmode=disable" \
  go run ./cmd/main.go
```

Or set environment variables in your shell from `.env`:
```bash
export $(grep -v '^#' .env | xargs)
cd services/ledger && go run ./cmd/main.go
```

Services listen on ports as defined in SYSTEM_ARCHITECTURE.md:
- gateway: `:8000`
- auth: `:8010`
- profiles: `:8020`
- ledger: `:8030`
- wallet: `:8040`
- payments: `:8050`
- ticketing: `:8060`
- orders: `:8070`
- catalog: `:8080`
- devices: `:8090`
- sessions: `:8100`
- reporting: `:8110`
- sync: `:8120`
- edge: `:9000`

---

## Building All Services

```bash
make build
# equivalent to: go build ./services/... ./edge/... ./pkg/...
```

Compiled binaries are written to each service's directory. They are gitignored.

---

## Running Tests

```bash
make test
# equivalent to: go test ./services/... ./edge/... ./pkg/...
```

Tests run against the local dev stack (Postgres + Redis must be up). Start with `make dev-up` first.

---

## Running Migrations Manually

Migrations run automatically when `make dev-up` is called. To run them separately:

```bash
make migrate
```

This executes `scripts/migrate.sh` which:
1. Runs `migrations/000_init_schemas.sql` (creates all service schemas)
2. Runs any `.sql` files in `migrations/{schema}/` subdirectories via golang-migrate

---

## Seeding the Database

Seed a test venue and user for development:

```bash
make seed
```

This executes `scripts/seed.sh`.

---

## Stopping the Dev Stack

```bash
make dev-down
# equivalent to: docker compose -f infra/docker-compose.dev.yml down
```

Postgres data persists in the `pgdata` Docker volume. To wipe it:

```bash
docker compose -f infra/docker-compose.dev.yml down -v
```

---

## Linting

Install golangci-lint: https://golangci-lint.run/usage/install/

```bash
make lint
# equivalent to: golangci-lint run ./services/... ./edge/... ./pkg/...
```

---

## Troubleshooting

### `go build` fails: cannot find module providing package github.com/golang-jwt/jwt/v5

Run `go work sync` from the repo root. This fetches and pins external module checksums.

### Postgres refuses connection

The container may still be starting. Wait 5-10 seconds and retry. Check health:
```bash
docker compose -f infra/docker-compose.dev.yml exec postgres pg_isready -U niteos
```

### Migration fails: role "niteos_app" already exists

The init schema migration uses `IF NOT EXISTS` for the role. This is safe to re-run. If you see other migration errors, check the migration files for syntax issues.

### Keys missing: jwt_private_key.pem not found

Run the key generation step from First-Time Setup (step 2 above). The `infra/secrets/` directory is gitignored and not populated on clone.

### Port conflict

Another process is using 5432 or 6379. Stop it, or change the port mapping in `infra/docker-compose.dev.yml` and update your `.env` accordingly.

---

## Environment variable reference

See `.env.example` for all variables with descriptions.

For production secret management, see `docs/SECRET_ROTATION.md`.
