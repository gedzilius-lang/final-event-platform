# Workspace-aware multi-stage builder for all NiteOS Go cloud services.
# ARG SERVICE = service directory name under services/ (e.g. gateway, auth, ledger …)
#
# Build (from repo root):
#   docker build --build-arg SERVICE=gateway -t niteos-gateway .
#
# docker compose cloud passes SERVICE via build.args — see infra/docker-compose.cloud.yml.

FROM golang:1.22-alpine AS builder
ARG SERVICE
RUN test -n "${SERVICE}" || (echo "ERROR: SERVICE build arg is required" && exit 1)

WORKDIR /workspace

# Copy workspace config before source for better layer caching.
COPY go.work go.work.sum ./

# All workspace members must be present for go.work.sum verification.
COPY pkg/     ./pkg/
COPY services/ ./services/
COPY edge/    ./edge/

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /service ./services/${SERVICE}/cmd/...

# Minimal runtime image — no shell, no package manager.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /service /service
ENTRYPOINT ["/service"]
