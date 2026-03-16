FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/orders ./cmd/

FROM scratch
COPY --from=builder /bin/orders /bin/orders
EXPOSE 80
ENTRYPOINT ["/bin/orders"]
