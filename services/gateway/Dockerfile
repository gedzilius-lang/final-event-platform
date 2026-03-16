# NOTE: This per-service Dockerfile is for local/standalone reference only.
# For cloud deployment, use the workspace-aware root Dockerfile:
#   docker build --build-arg SERVICE=gateway -t niteos-gateway .
# docker-compose.cloud.yml uses the root Dockerfile automatically.

FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/gateway ./cmd/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/gateway /bin/gateway
EXPOSE 8000
ENTRYPOINT ["/bin/gateway"]
