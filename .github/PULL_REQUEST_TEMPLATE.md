## Summary

<!-- Briefly describe what this pull request changes and why. -->

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Refactor / cleanup
- [ ] Documentation
- [ ] Infrastructure / CI

## Verification

- [ ] Backend (`backend/`): `gofmt -l .` clean, `go vet ./...`, `go test -race ./...`, `go build ./cmd/server/main.go`
- [ ] Frontend (`frontend/`): `npx prettier --check .`, `npm run lint`, `npm run build`
- [ ] Database migrations tested (if applicable)
- [ ] OpenAPI / Swagger docs updated (if API routes changed)
- [ ] No secrets, `.env` files, or local artifacts included
