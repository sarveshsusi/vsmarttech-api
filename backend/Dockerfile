# ================================
# Stage 1: Build
# ================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker-sla ./cmd/worker-sla \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker-contracts ./cmd/worker-contracts

# ================================
# Stage 2: Runtime (shared image, different CMD per service)
# ================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

COPY --from=builder /out/api /out/worker-sla /out/worker-contracts ./

EXPOSE 8080

# Default process is the HTTP API; Compose overrides CMD for workers.
CMD ["./api"]
