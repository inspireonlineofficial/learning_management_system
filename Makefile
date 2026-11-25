.PHONY: help backend-run backend-build backend-test backend-vet backend-fmt \
        frontend-install frontend-dev frontend-build frontend-lint frontend-format frontend-format-check \
        docker-up docker-down docker-build gen-keys mailsink storage-cors email-check

help:
	@echo "Inspire LMS - Available targets:"
	@echo "  backend-run           Run Go API server locally"
	@echo "  backend-build         Build Go API binary into ./backend/bin/server"
	@echo "  backend-test          Run Go test suite with race detector"
	@echo "  backend-vet           Run go vet across all backend packages"
	@echo "  backend-fmt           Format Go source files with gofmt"
	@echo "  frontend-install      Install frontend dependencies with npm ci"
	@echo "  frontend-dev          Start frontend Vite dev server"
	@echo "  frontend-build        Build frontend for production"
	@echo "  frontend-lint         Run ESLint on frontend"
	@echo "  frontend-format       Format frontend files with Prettier"
	@echo "  frontend-format-check Check frontend formatting with Prettier"
	@echo "  docker-build          Build Docker Compose images"
	@echo "  docker-up             Start full stack via Docker Compose"
	@echo "  docker-down           Stop Docker Compose services"
	@echo "  gen-keys              Generate local RS256 JWT key pair in ./keys"
	@echo "  mailsink              Run the development mail server (inbox http://127.0.0.1:8025)"
	@echo "  storage-cors          Show/set the storage CORS rule (ARGS=\"-origins https://... -apply\")"
	@echo "  email-check           Check SMTP login and SPF/DKIM/DMARC (ARGS=\"-to you@example.com\" sends a test)"

backend-run:
	cd backend && go run ./cmd/server/main.go

backend-build:
	cd backend && go build -o ./bin/server ./cmd/server/main.go

backend-test:
	cd backend && go test -v -race ./...

backend-vet:
	cd backend && go vet ./...

backend-fmt:
	cd backend && gofmt -w .

frontend-install:
	cd frontend && npm ci

frontend-dev:
	cd frontend && npm run dev

frontend-build:
	cd frontend && npm run build

frontend-lint:
	cd frontend && npm run lint

frontend-format:
	cd frontend && npm run format

frontend-format-check:
	cd frontend && npx prettier --check .

mailsink:
	cd backend && go run ./cmd/mailsink

storage-cors:
	cd backend && go run ./cmd/storagecors $(ARGS)

email-check:
	cd backend && go run ./cmd/emailcheck $(ARGS)

docker-build:
	docker compose --env-file .env build

docker-up:
	docker compose --env-file .env up -d

docker-down:
	docker compose down

gen-keys:
	mkdir -p keys
	openssl genrsa -out keys/private.pem 2048
	openssl rsa -in keys/private.pem -pubout -out keys/public.pem
