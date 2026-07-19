# ================================
# Stage 1: Build
# ================================
FROM golang:1.25-alpine AS builder

# Pull patched toolchain matching go.mod (fixes stdlib vulns flagged by govulncheck)
ENV GOTOOLCHAIN=go1.25.12

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Cache module download separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/worker-sla ./cmd/worker-sla \
 && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/worker-contracts ./cmd/worker-contracts

# ================================
# Stage 2: Runtime (shared image, different CMD per service)
# ================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl \
 && addgroup -S app && adduser -S -G app app \
 && mkdir -p /app && chown -R app:app /app

WORKDIR /app

COPY --from=builder --chown=app:app /out/api /out/worker-sla /out/worker-contracts ./

USER app

EXPOSE 8080

# Default process is the HTTP API; Compose overrides CMD for workers.
CMD ["./api"]
