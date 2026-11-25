# Contributing to Inspire LMS

Thank you for contributing to Inspire LMS. Please follow these guidelines to keep the codebase consistent and reliable.

## Development Workflow

1. Create a focused branch from `main` (e.g. `feat/course-notes-filter` or `fix/hls-fallback`).
2. Keep pull requests scoped to a single feature, bug fix, or refactor.
3. Run all relevant checks before opening a PR.

## Backend Guidelines (`backend/`)

- Follow the Clean Architecture layer boundaries:
  - `backend/internal/domain/` must not depend on `application`, `infrastructure`, or `interfaces`.
  - `backend/internal/application/` orchestrates business logic through domain interfaces.
  - `backend/internal/infrastructure/` implements storage, external APIs, and background workers.
  - `backend/internal/interfaces/http/` handles HTTP parsing, serialization, and middleware only.
- Run commands from `backend/`: format with `gofmt -w .`, verify `go vet ./...`, and run `go test -v -race ./...`.
- **Database migrations**: Always add new numbered `.up.sql` and `.down.sql` pairs in `backend/migrations/`. Never modify or rename already-applied migration files.
- **API documentation**: When adding or modifying HTTP endpoints, update Swagger annotations and regenerate `backend/docs/` so `swagger_spec_properties_test.go` passes.

## Frontend Guidelines (`frontend/`)

- Use `npm` with `frontend/package-lock.json` (CI runs `npm ci`).
- Place API calls in typed adapters under `frontend/src/lib/api/` rather than calling `fetch` directly inside route components.
- Ensure Prettier formatting (`npx prettier --check .`), ESLint (`npm run lint`), and production builds (`npm run build`) pass inside `frontend/`.

## Security & Secrets

- Never commit `.env`, private keys (`*.pem`, `*.key`), or real credentials.
- CI runs Gitleaks on every push and pull request; ensure no sensitive data enters Git history.
