FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/ticketing ./cmd/

FROM scratch
COPY --from=builder /bin/ticketing /bin/ticketing
EXPOSE 80
ENTRYPOINT ["/bin/ticketing"]
