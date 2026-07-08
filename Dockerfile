# ---- Build stage ----
FROM golang:1.23-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary, no CGO, stripped symbols for a smaller artifact.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/stress-test .

# ---- Final stage ----
FROM scratch

# TLS root certificates so HTTPS targets work out of the box.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/stress-test /usr/local/bin/stress-test

ENTRYPOINT ["/usr/local/bin/stress-test"]
