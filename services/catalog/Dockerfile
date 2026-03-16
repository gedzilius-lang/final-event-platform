FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/catalog ./cmd/

FROM scratch
COPY --from=builder /bin/catalog /bin/catalog
EXPOSE 80
ENTRYPOINT ["/bin/catalog"]
