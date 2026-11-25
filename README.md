# Inspire — Learning Management System

Inspire is a full-stack Learning Management System (LMS) organized into dedicated **`backend/`** (Go Clean Architecture REST API) and **`frontend/`** (TanStack Start / React 19) applications. It supports student learning workflows, teacher course authoring and grading, and platform administration across courses, assessments, live classes, a digital/physical bookshop, community forums, gamification, and analytics.

## Key Features

- **Authentication & Authorization** — Email/password registration with OTP verification, RS256 JWT access/refresh tokens with JWKS endpoint (`/.well-known/jwks.json`), Google OAuth 2.0, password reset, admin two-step OTP login, role-based access control (RBAC), and append-only audit logging.
- **Course Authoring & Delivery** — Multi-level curriculum hierarchy (courses → modules → chapters → lessons), free/paid course pricing, teacher course submission & admin review workflow, course notes, student reviews, comments, and course progress tracking.
- **Video Pipeline & Streaming** — Direct and resumable multipart uploads to private S3-compatible storage (videos are playable immediately as MP4). A background FFmpeg worker builds an adaptive HLS ladder (240p–1080p, capped at the source resolution, aligned 4 s segments). Playback authorization issues short-lived signed URLs: backend-signed playlists by default, or an optional token-authenticated CDN. The hls.js player adds adaptive and manual quality, resume, offline recovery and throttled progress sync.
- **Assessments & Assignments** — Timed quizzes with draft auto-save, multi-attempt tracking, automatic grading, and assignments with file/text submissions and teacher grading.
- **Live Classes** — Scheduled teacher live sessions, timezone-aware student calendars, room join tokens, participant moderation, chat, attendance tracking, and recording controls.
- **Bookshop & Purchase Approvals** — Public and student book catalog (physical, digital, or combined formats), cover uploads, cart & orders, digital reader access with bookmarks, student purchase requests, admin approval workflows, and idempotent refund processing.
- **Community Forum & Moderation** — Markdown posts (XSS-sanitized via Bluemonday), pre-publication admin review queue, comments, upvotes, and user-flagged content moderation.
- **Gamification & Certificates** — Configurable point awards, student points history, weekly/all-time leaderboards with opt-out privacy controls, and verifiable course completion certificates.
- **Search, Notifications & Analytics** — Full-text global search powered by Typesense, notification inbox/templates/broadcasts, hourly analytics pre-aggregation, student/teacher/admin dashboards, promotional homepage slides, and versioned system settings with rollback and maintenance mode.

## Technology Stack

| Layer | Technologies |
| --- | --- |
| **Backend API (`backend/`)** | Go 1.25 (`net/http`), Clean Architecture (`domain` / `application` / `infrastructure` / `interfaces`), `swaggo/swag` OpenAPI docs |
| **Frontend (`frontend/`)** | TypeScript, React 19, TanStack Start, TanStack Router, TanStack Query, Tailwind CSS v4, Radix UI / shadcn/ui, Vite, Nitro (`node-server`) |
| **Data & Search** | PostgreSQL 16 (46 SQL migrations, numbered 001–045), Redis 7 (sessions, rate limiting, caching, leaderboards, job queues), Typesense 27 |
| **Media & Storage** | S3-compatible object storage (DigitalOcean Spaces / RustFS via AWS SDK for Go), FFmpeg (HLS transcoding) |
| **Infrastructure (`infra/`)** | Docker, Docker Compose, Nginx, GitHub Actions CI/CD, Gitleaks secret scanning |

## Architecture Overview

The backend (`backend/`) follows a modular monolith Clean Architecture organized around bounded contexts (`analytics`, `assessments`, `audit`, `auth`, `bookshop`, `certificates`, `courses`, `enrollments`, `forum`, `live_sessions`, `notifications`, `payments`, `points`, `rbac`, `search`, `slides`, `system_config`, `users`). All API endpoints are versioned under `/v1/` with a global middleware stack handling request IDs, structured JSON logging with secret redaction, maintenance mode, security headers, CORS, and Redis-backed rate limiting.

The frontend (`frontend/`) is a TanStack Start SSR/SPA application communicating with the Go API through typed adapters under `frontend/src/lib/api/` and role-scoped route layouts for public visitors, students, teachers, and administrators.

## Project Structure

```text
.
├── backend/                     # Go REST API server & background workers
│   ├── cmd/
│   │   ├── server/              # API entrypoint and dependency wiring
│   │   └── …                    # Dev/ops tools: mailsink, emailcheck, storagecors, bookshopload
│   ├── configs/                 # Environment variable loading and validation
│   ├── docs/                    # Generated Swagger/OpenAPI spec (docs.go, swagger.json, swagger.yaml)
│   ├── internal/
│   │   ├── application/         # Use-case services, commands, DTOs, and unit/property tests
│   │   ├── domain/              # Domain entities and repository interfaces
│   │   ├── infrastructure/      # PostgreSQL, Redis, Typesense, RustFS/S3, JWT, OAuth, SMTP, FFmpeg, workers
│   │   ├── integration/         # Cross-context in-memory integration tests
│   │   └── interfaces/http/     # HTTP server, route registration, handlers, and middleware
│   ├── migrations/              # Versioned PostgreSQL up/down SQL migrations (001–045)
│   ├── pkg/                     # Shared Go packages (apperrors, logger, pagination, validator)
│   ├── .dockerignore
│   ├── Dockerfile               # Multi-stage Go + Alpine/FFmpeg image
│   ├── go.mod
│   └── go.sum
├── frontend/                    # TanStack Start / React 19 web application
│   ├── scripts/                 # Playwright student route audit script
│   ├── src/
│   │   ├── assets/              # Static brand and UI images
│   │   ├── components/          # Domain and shadcn/ui components
│   │   ├── context/             # Auth context provider
│   │   ├── hooks/               # Custom React hooks
│   │   ├── lib/                 # API client, typed endpoint adapters, upload & session helpers
│   │   └── routes/              # File-based TanStack Router routes (public, student, teacher, admin)
│   ├── Dockerfile               # Multi-stage Node 22 + Nitro image
│   ├── package.json
│   └── package-lock.json
├── infra/
│   └── nginx/                   # Production HTTPS and HTTP fallback Nginx virtual host configs
├── .github/                     # CI/CD workflow and GitHub issue/PR templates
├── docker-compose.yml           # Full stack (PostgreSQL, Redis, Typesense, Backend, Frontend)
├── Makefile                     # Common build, test, lint, and Docker commands
└── .env.example                 # Environment variable template
```

## Prerequisites

- **Docker & Docker Compose** (recommended for running the full stack)
- For local development without containers:
  - **Go 1.25+**
  - **Node.js 22+** and **npm**
  - **PostgreSQL 16**, **Redis 7**, **Typesense 27**, and an **S3-compatible object store**
  - **FFmpeg** (for background HLS video transcoding)
  - **OpenSSL** (to generate local RS256 JWT keys)

## Getting Started

### 1. Configure Environment Variables

Copy `.env.example` to `.env` and fill in your local credentials and object storage configuration:

```bash
cp .env.example .env
```

### 2. Generate RS256 JWT Keys

The backend signs JWTs using an RSA key pair mounted at `./keys/`:

```bash
make gen-keys
# Or manually:
mkdir -p keys
openssl genrsa -out keys/private.pem 2048
openssl rsa -in keys/private.pem -pubout -out keys/public.pem
```

### 3. Run with Docker Compose

```bash
docker compose --env-file .env up -d --build
```

Services exposed locally:

- **Frontend**: `http://localhost:3000`
- **Backend API**: `http://localhost:8080`
- **Health Check**: `http://localhost:8080/health`
- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Typesense**: `http://127.0.0.1:8108`

Database migrations in `backend/migrations/` are applied automatically by the Go server on startup.

## Local Development

### Backend (`backend/`)

```bash
cd backend
gofmt -w .
go vet ./...
go test -v -race ./...
go build -o ./bin/server ./cmd/server/main.go

# Run locally (requires .env variables exported in your shell)
go run ./cmd/server/main.go
```

### Frontend (`frontend/`)

```bash
cd frontend
npm ci
npm run dev       # Start Vite dev server
npm run lint      # Run ESLint
npx prettier --check .
npm run build     # Build production Nitro node-server bundle
```

## Environment Variables

All supported variables are documented in [.env.example](.env.example). Required backend startup variables (validated in `backend/configs/config.go`):

| Variable | Description |
| --- | --- |
| `DATABASE_DSN` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection URL |
| `RUSTFS_ENDPOINT` | S3-compatible object storage endpoint |
| `RUSTFS_ACCESS_KEY` / `RUSTFS_SECRET_KEY` | Object storage credentials |
| `JWT_PRIVATE_KEY_PATH` / `JWT_PUBLIC_KEY_PATH` | Paths to RS256 PEM key files |
| `TYPESENSE_HOST` / `TYPESENSE_API_KEY` | Typesense search server host and API key |

Video delivery: `HLS_DELIVERY_MODE` (`presigned` default, or `bunny`) with `BUNNY_CDN_HOST` / `BUNNY_TOKEN_KEY`.

Optional variables include `SERVER_PORT`, `FRONTEND_BASE_URL`, `RUSTFS_REGION`, bucket names (`RUSTFS_*_BUCKET`), `JWT_ISSUER`, SMTP settings (`SMTP_*`), Google OAuth settings (`GOOGLE_*`, `OAUTH_TOKEN_ENCRYPTION_KEY`), `ADMIN_DEV_BYPASS` / `ADMIN_DEV_OTP` (local development only), and `VITE_API_BASE_URL`.

## Testing & Verification

CI runs on every push and pull request to `main` via [.github/workflows/ci.yml](.github/workflows/ci.yml):

1. **Secret scanning** — Gitleaks across full repository history.
2. **Backend CI (`./backend`)** — `gofmt -l .`, `go vet ./...`, `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...`, and `go build ./cmd/server/main.go`.
3. **Frontend CI (`./frontend`)** — `npm ci`, `npx prettier --check .`, `npm run lint`, and `npm run build`.
4. **Docker build verification** — Multi-stage builds for both `./backend/Dockerfile` and `./frontend/Dockerfile`.

Locally:

```bash
cd backend && go test ./...      # unit + property tests; FFmpeg integration test runs when ffmpeg is installed
cd frontend && npm run build && npm run lint
```

## Known Limitations

- Admin accounts sign in with a password only. The API supports an admin OTP step, but the web UI does not use it yet.
- Access and refresh tokens are stored in `localStorage`.
- There is no payment gateway: paid courses and books use admin-approved purchase requests.
- Signed playback URLs stay valid for up to 2 hours after an enrollment is revoked.

## Deployment

Production deployment runs on a Linux host with Docker Compose and Nginx reverse proxy (HTTPS via Let's Encrypt / Certbot) automated by the `deploy-production` job in [.github/workflows/ci.yml](.github/workflows/ci.yml).

## Contributing & Security

- Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.
- To report a security vulnerability, follow [SECURITY.md](SECURITY.md).
