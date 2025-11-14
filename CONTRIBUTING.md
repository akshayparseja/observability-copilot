# Contributing

## Setup

```bash
# Backend
cd backend && go mod download && go run ./cmd/server/main.go

# Frontend
cd copilot-ui && npm install && npm start

# Database
docker run -d -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=copilot -p 5432:5432 postgres:13
```

## Workflow

1. Fork & clone
2. `git checkout -b feat/feature-name`
3. Make changes
4. `git commit -m "feat: description"`
5. Push & open PR

## Commit Format

```
feat(scope): description
fix(scanner): timeout
docs(readme): update
test(api): tests
```

## Code Style

- **Go**: `go fmt ./...`
- **React**: Functional components, meaningful names
- **Comments**: Explain "why"

## Test

```bash
cd backend && go test ./...
cd copilot-ui && npm test
```

## Help With

- Add Node.js, .NET, Rust frameworks
- Improve scanner patterns
- Authentication & API keys
- UI improvements & dark mode
- Test coverage

---

Questions? Check [README.md](README.md) or open an issue.
