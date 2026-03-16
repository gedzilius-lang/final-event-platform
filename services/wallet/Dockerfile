FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/wallet ./cmd/

FROM scratch
COPY --from=builder /bin/wallet /bin/wallet
EXPOSE 80
ENTRYPOINT ["/bin/wallet"]
