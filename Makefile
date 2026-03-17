.PHONY: build test lint dev-up dev-down migrate seed admin-dev admin-build \
        guest-dev guest-build \
        cloud-build cloud-up cloud-down cloud-logs cloud-ps cloud-migrate help

# Default target
help:
	@echo "NiteOS build targets:"
	@echo ""
	@echo "  Development:"
	@echo "    make build         — build all Go services and edge binary"
	@echo "    make test          — run all Go tests"
	@echo "    make lint          — run golangci-lint on all modules"
	@echo "    make dev-up        — start local Postgres + Redis via Docker Compose"
	@echo "    make dev-down      — stop local dev stack"
	@echo "    make migrate       — run pending Postgres migrations (requires dev stack)"
	@echo "    make seed          — seed local database with test data"
	@echo "    make admin-dev     — run Next.js admin console in dev mode"
	@echo "    make admin-build   — production build of admin console"
	@echo ""
	@echo "  Cloud deployment (VPS A):"
	@echo "    make cloud-build   — build all Docker images for production"
	@echo "    make cloud-up      — start cloud stack (requires infra/cloud.env)"
	@echo "    make cloud-down    — stop cloud stack"
	@echo "    make cloud-logs    — tail all service logs"
	@echo "    make cloud-ps      — show running services"
	@echo "    make cloud-migrate    — run SQL migrations against cloud Postgres"
	@echo "    make cloud-preflight  — validate env/secrets before deploying"
	@echo "    make cloud-healthcheck — check all service healthz endpoints"
	@echo "    make cloud-smoketest  — pilot smoke test (requires live stack)"

build:
	go build ./services/... ./edge/... ./pkg/...

test:
	go test ./services/... ./edge/... ./pkg/...

lint:
	golangci-lint run ./services/... ./edge/... ./pkg/...

dev-up:
	docker compose -f infra/docker-compose.dev.yml up -d
	@echo "Waiting for Postgres to be ready..."
	@docker compose -f infra/docker-compose.dev.yml exec postgres pg_isready -U niteos || sleep 3
	@echo "Dev stack ready. Postgres: localhost:5432  Redis: localhost:6379"

dev-down:
	docker compose -f infra/docker-compose.dev.yml down

migrate:
	@echo "Running migrations..."
	./scripts/migrate.sh

seed:
	@echo "Seeding database with test venue and user..."
	./scripts/seed.sh

admin-dev:
	cd web/admin && npm install && npm run dev

admin-build:
	cd web/admin && npm install && npm run build

guest-dev:
	cd web/guest && npm install && npm run dev

guest-build:
	cd web/guest && npm install && npm run build

# ── Cloud deployment targets (NiteOS VPS) ─────────────────────────────────────
CLOUD_COMPOSE = docker compose -f infra/docker-compose.cloud.yml --env-file infra/cloud.env

cloud-build:
	$(CLOUD_COMPOSE) build --parallel

cloud-up:
	$(CLOUD_COMPOSE) up -d

cloud-down:
	$(CLOUD_COMPOSE) down

cloud-logs:
	$(CLOUD_COMPOSE) logs -f --tail=100

cloud-ps:
	$(CLOUD_COMPOSE) ps

cloud-migrate:
	@echo "Running migrations against cloud Postgres..."
	DATABASE_URL="postgres://niteos:$$(grep POSTGRES_PASSWORD infra/cloud.env | cut -d= -f2)@localhost:5432/niteos?sslmode=disable" \
	  ./scripts/migrate.sh

cloud-preflight:
	@bash scripts/preflight-cloud.sh

cloud-healthcheck:
	@bash scripts/healthcheck-cloud.sh

cloud-smoketest:
	@bash scripts/smoke-test-pilot.sh
