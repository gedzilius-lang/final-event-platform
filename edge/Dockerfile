FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/edge ./cmd/

FROM alpine:3.19
RUN apk add --no-cache sqlite
COPY --from=builder /bin/edge /bin/edge
EXPOSE 9000
ENTRYPOINT ["/bin/edge"]
