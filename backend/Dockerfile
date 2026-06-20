# ================================
# Stage 1: Build
# ================================
FROM golang:1.25-alpine AS builder

# Install build dependencies (needed for CGO-free postgres driver)
RUN apk add --no-cache git

WORKDIR /build

# Cache dependency downloads separately from source
COPY go.mod go.sum ./
RUN go mod download

# Copy full source
COPY . .

# Build binary from ./main package
# CGO_ENABLED=0 produces a fully static binary for the minimal runtime image
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./main

# ================================
# Stage 2: Runtime
# ================================
FROM alpine:3.21

# ca-certificates: needed for HTTPS calls (AWS S3, Mailtrap, Supabase)
# tzdata: needed for cron jobs / timezone-aware scheduling
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/server .

EXPOSE 8080

CMD ["./server"]
