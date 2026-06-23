# Stage 1: Builder
FROM golang:1.25.11-alpine AS builder

# Install CA certificates for TLS (Requirement 5.1)
RUN apk --no-cache add ca-certificates

# Create non-root system user with UID 1001 (Requirement 4.1)
RUN adduser -D -u 1001 appuser

WORKDIR /build

# Use writable build paths inside the container build environment.
ENV GOCACHE=/build/go-build-cache
ENV GOTMPDIR=/build/go-tmp
RUN mkdir -p /build/go-build-cache /build/go-tmp

# Copy dependency manifests first for layer caching (Requirements 3.1, 3.3)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy full source and compile static binary (Requirements 2.1, 2.2, 2.3, 2.4)
COPY . .
RUN --mount=type=cache,target=/build/go-build-cache \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/server ./cmd/server/main.go

# Stage 2: Final image
FROM scratch

# Copy CA certificates for outbound HTTPS (Requirement 5.2)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy passwd for non-root user resolution (Requirement 4.2)
COPY --from=builder /etc/passwd /etc/passwd

# Copy compiled binary
COPY --from=builder /app/server /app/server

# Copy SQL migration files (Requirement 1.3)
COPY --from=builder /build/migrations /app/migrations

# Use /app so relative paths like "migrations" resolve correctly at runtime.
WORKDIR /app

# Run as non-root user (Requirement 4.3)
USER appuser

# Default port (Requirement 9.4)
ENV SERVER_PORT=8080
EXPOSE 8080

# Health check (Requirements 7.1, 7.2)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/server", "healthcheck"]

ENTRYPOINT ["/app/server"]
