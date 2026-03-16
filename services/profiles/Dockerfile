FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/profiles ./cmd/

FROM scratch
COPY --from=builder /bin/profiles /bin/profiles
EXPOSE 80
ENTRYPOINT ["/bin/profiles"]
