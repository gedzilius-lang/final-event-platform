FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/devices ./cmd/

FROM scratch
COPY --from=builder /bin/devices /bin/devices
EXPOSE 80
ENTRYPOINT ["/bin/devices"]
