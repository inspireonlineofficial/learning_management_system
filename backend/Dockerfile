# Stage 1: Builder
FROM golang:1.25.11-alpine AS builder

RUN apk --no-cache add ca-certificates

RUN adduser -D -u 1001 appuser

WORKDIR /build

# Use writable build paths inside the container build environment.
ENV GOCACHE=/build/go-build-cache
ENV GOTMPDIR=/build/go-tmp
RUN mkdir -p /build/go-build-cache /build/go-tmp

# Copy dependency manifests first for layer caching.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source and compile a static Linux binary.
COPY . .
RUN --mount=type=cache,target=/build/go-build-cache \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/server ./cmd/server

# Stage 2: Runtime image
FROM alpine:3.21

RUN apk --no-cache add ca-certificates ffmpeg

RUN adduser -D -u 1001 appuser

COPY --from=builder /app/server /app/server
COPY --from=builder /build/migrations /app/migrations

# Run from /app so relative paths like "migrations" resolve correctly at runtime.
WORKDIR /app

USER appuser

ENV SERVER_PORT=8080
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/server", "healthcheck"]

ENTRYPOINT ["/app/server"]
