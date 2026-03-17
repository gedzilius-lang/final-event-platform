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

# Alpine runtime — minimal footprint with shell+wget for healthchecks.
FROM alpine:3.19
RUN apk add --no-cache wget ca-certificates \
 && addgroup -S niteos \
 && adduser  -S -G niteos niteos
COPY --from=builder /service /service
USER niteos
ENTRYPOINT ["/service"]
