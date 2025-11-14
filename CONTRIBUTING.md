# Contributing to Observability Copilot 🚀

Thank you for your interest in contributing to Observability Copilot! We're excited to have you join our community. This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Areas for Contribution](#areas-for-contribution)
- [Reporting Issues](#reporting-issues)
- [Project Structure](#project-structure)
- [Testing](#testing)
- [Code Style](#code-style)
- [Documentation](#documentation)

## Code of Conduct

We are committed to providing a welcoming and inclusive environment for all contributors. Please note that this project is released with a [Contributor Code of Conduct](CODE_OF_CONDUCT.md). By participating in this project, you agree to abide by its terms.

### Expected Behavior
- Use welcoming and inclusive language
- Be respectful of differing opinions and experiences
- Accept constructive criticism gracefully
- Focus on what is best for the community
- Show empathy towards other community members

### Unacceptable Behavior
- Harassment, discrimination, or intimidation
- Offensive comments related to gender, sexuality, race, or religion
- Deliberate intimidation or bullying
- Unwelcome sexual advances or attention
- Sharing others' private information without consent

## Getting Started

### Prerequisites

Before you start contributing, ensure you have:

- **Git** - Version control (https://git-scm.com/)
- **Go 1.21+** - Backend development (https://golang.org/)
- **Node.js 18+** - Frontend development (https://nodejs.org/)
- **Docker** - For containerized development (https://www.docker.com/)
- **PostgreSQL 13+** - Database (or use Docker)
- **GitHub Account** - For creating PRs (https://github.com)

### Fork and Clone

1. **Fork the repository** on GitHub:
   ```bash
   # Visit https://github.com/akshayparseja/observability-copilot
   # Click "Fork" button in top-right corner
   ```

2. **Clone your fork locally:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/observability-copilot.git
   cd observability-copilot
   ```

3. **Add upstream remote:**
   ```bash
   git remote add upstream https://github.com/akshayparseja/observability-copilot.git
   ```

4. **Verify remotes:**
   ```bash
   git remote -v
   # origin    https://github.com/YOUR_USERNAME/observability-copilot.git (fetch)
   # origin    https://github.com/YOUR_USERNAME/observability-copilot.git (push)
   # upstream  https://github.com/akshayparseja/observability-copilot.git (fetch)
   # upstream  https://github.com/akshayparseja/observability-copilot.git (push)
   ```

## Development Setup

### Backend Setup

```bash
# Install dependencies
cd backend
go mod download
go mod tidy

# Create .env file (optional)
cat > .env << 'EOF'
DATABASE_URL=postgres://postgres:postgres@localhost:5432/copilot?sslmode=disable
GITHUB_TOKEN=your_github_token_here
PORT=8000
EOF

# Start PostgreSQL (if not already running)
docker run -d \
  --name copilot-db \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=copilot \
  -p 5432:5432 \
  postgres:13

# Run backend server
go run ./cmd/server/main.go
# Server will start on http://localhost:8000
```

### Frontend Setup

```bash
# Install dependencies
cd copilot-ui
npm install

# Start development server
npm start
# Frontend will open at http://localhost:3000
```

### Run Both Together

Using Docker Compose (recommended):

```bash
# Create docker-compose.yml in project root (if not present)
cat > docker-compose.yml << 'EOF'
version: '3.8'
services:
  postgres:
    image: postgres:13
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: copilot
    ports:
      - "5432:5432"

  backend:
    build: ./backend
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/copilot?sslmode=disable
      GITHUB_TOKEN: ${GITHUB_TOKEN}
      PORT: 8000
    ports:
      - "8000:8000"
    depends_on:
      - postgres

  frontend:
    build: ./copilot-ui
    ports:
      - "3000:3000"
    environment:
      REACT_APP_API_URL: http://localhost:8000/api
    depends_on:
      - backend
EOF

# Start all services
GITHUB_TOKEN="your_token_here" docker-compose up
```

## Making Changes

### Create a Feature Branch

Always create a new branch for each feature or bugfix:

```bash
# Update your local main branch
git fetch upstream
git checkout main
git rebase upstream/main

# Create a new feature branch
git checkout -b feat/your-feature-name
# or for bugfixes:
git checkout -b fix/your-bugfix-name
```

### Branch Naming Conventions

Use one of these prefixes:

- `feat/` - New feature (e.g., `feat/node-instrumentation`)
- `fix/` - Bug fix (e.g., `fix/scanner-timeout`)
- `docs/` - Documentation updates (e.g., `docs/add-troubleshooting`)
- `test/` - Test additions (e.g., `test/scanner-unit-tests`)
- `chore/` - Maintenance tasks (e.g., `chore/update-dependencies`)
- `refactor/` - Code refactoring (e.g., `refactor/generator-structure`)
- `perf/` - Performance improvements (e.g., `perf/optimize-clone`)

### Make Your Changes

Edit the relevant files. See [Project Structure](#project-structure) for file organization.

## Commit Guidelines

### Commit Message Format

Follow the Conventional Commits specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Examples

```bash
# Feature
git commit -m "feat(scanner): add timeout for git clone operations"

# Bug fix
git commit -m "fix(generator): correct Python dependency version"

# Documentation
git commit -m "docs(readme): add Kubernetes deployment section"

# Test
git commit -m "test(scanner): add framework detection tests"

# Chore
git commit -m "chore: update go dependencies"
```

### Commit Message Rules

1. **Type** - One of: `feat`, `fix`, `docs`, `test`, `chore`, `refactor`, `perf`
2. **Scope** (optional) - Component affected: `scanner`, `generator`, `github`, `api`, `ui`, `db`
3. **Subject**
   - Use imperative mood ("add" not "adds" or "added")
   - Don't capitalize first letter
   - No period at the end
   - Limit to 50 characters
4. **Body** (optional)
   - Explain the "why", not the "what"
   - Limit each line to 72 characters
   - Separate from subject with blank line
5. **Footer** (optional)
   - Reference issues: `Fixes #123`
   - Note breaking changes: `BREAKING CHANGE: description`

### Example Commit with Details

```bash
git commit -m "feat(scanner): add timeout for repository cloning

Previously, scanning very large repositories could hang indefinitely.
This change adds a configurable timeout (default 5 minutes) to the
git clone operation.

- Add context.WithTimeout to scanner.ScanRepo
- Make timeout configurable via environment variable
- Add tests for timeout behavior

Fixes #45"
```

## Pull Request Process

### Before Creating a PR

1. **Sync with upstream:**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run tests locally:**
   ```bash
   # Backend
   cd backend && go test ./...
   
   # Frontend
   cd copilot-ui && npm test
   ```

3. **Check code style:**
   ```bash
   # Go
   cd backend && go fmt ./...
   
   # TypeScript/React
   cd copilot-ui && npm run lint (if available)
   ```

4. **Push your branch:**
   ```bash
   git push origin feat/your-feature-name
   ```

### Creating the PR

1. **Go to GitHub** and click "Compare & pull request"
2. **Fill in the PR template:**
   ```markdown
   ## Description
   Brief description of changes
   
   ## Type of Change
   - [ ] New feature
   - [ ] Bug fix
   - [ ] Documentation update
   - [ ] Performance improvement
   
   ## Checklist
   - [ ] Code follows style guidelines
   - [ ] Tests added/updated
   - [ ] Documentation updated
   - [ ] Commits are well-formatted
   - [ ] No breaking changes (or documented)
   
   ## Testing
   How to test these changes
   
   ## Related Issues
   Fixes #123
   ```

3. **Wait for review** - Maintainers will review your PR

### PR Review Guidelines

- Be open to feedback
- Respond to comments promptly
- Make requested changes in new commits (don't rewrite history)
- Re-request review after making changes
- Keep discussions professional and constructive

### Merging

PRs are merged using "Squash and merge" to keep history clean. Your commits will be squashed into one with the PR title as commit message.

## Areas for Contribution

### High Priority

#### 1. Add Framework Support

**Current:** Go, Python, Java  
**Needed:** Node.js, .NET, Rust

**What to do:**
- Create `backend/pkg/generator/nodejs_generator.go`
- Implement `generateNodeInstrumentation()`
- Add Node.js patterns to `pkg/scanner/scanner.go`
- Test with real repositories
- Update documentation

**Complexity:** Medium  
**Time estimate:** 4-6 hours

#### 2. Improve Scanner Detection

**Current issues:**
- No timeout on `git clone`
- Limited instrumentation patterns
- Potential false positives

**What to do:**
- Add timeout context
- Expand pattern library
- Improve two-pass detection
- Add regression tests
- Profile performance

**Complexity:** Medium  
**Time estimate:** 3-4 hours

#### 3. Authentication & Authorization

**Current:** No authentication  
**Needed:** User auth, API keys, role-based access

**What to do:**
- Add JWT authentication
- Implement API key support
- Add user roles (admin, user, viewer)
- Protect endpoints
- Add tests

**Complexity:** High  
**Time estimate:** 8-10 hours

### Medium Priority

#### 4. Webhook Support

**Feature:** Auto-scan on repository events

**What to do:**
- Add GitHub webhook receiver
- Validate webhook signatures
- Queue async scanning
- Handle retries
- Add webhook management UI

**Complexity:** High  
**Time estimate:** 6-8 hours

#### 5. UI Improvements

**Needed:**
- Dark mode
- Repository filtering/search
- Batch operations
- Better error messages
- Loading states
- Mobile responsiveness

**Complexity:** Medium  
**Time estimate:** 3-5 hours per feature

#### 6. Testing

**Current:** Minimal tests  
**Needed:** Comprehensive test coverage

**Unit tests:**
```bash
cd backend
go test ./... -v
```

**Integration tests:**
- Test with real repositories
- Test with different frameworks
- Test error scenarios

**Frontend tests:**
```bash
cd copilot-ui
npm test
```

**Complexity:** Low-Medium  
**Time estimate:** 2-4 hours

### Lower Priority

#### 7. Documentation

- Add more examples
- Create video tutorials
- Improve error messages
- Add troubleshooting guide
- Document API endpoints
- Create architecture diagrams

#### 8. Performance

- Profile and optimize scanner
- Cache detection results
- Parallelize analysis
- Optimize database queries
- Add monitoring

#### 9. Security

- Add rate limiting
- Validate inputs
- Sanitize outputs
- Add CORS restrictions
- Document security practices

## Reporting Issues

### Before Opening an Issue

1. **Search existing issues** - Your issue might already be reported
2. **Check documentation** - Answer might be in README or docs
3. **Test with latest code** - Issue might be fixed already

### Issue Format

When creating an issue, include:

```markdown
## Description
Clear description of the issue

## Steps to Reproduce
1. Step 1
2. Step 2
3. ...

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Environment
- OS: macOS / Linux / Windows
- Go version: (if relevant)
- Node version: (if relevant)
- Browser: (if frontend issue)

## Screenshots/Logs
Attach relevant screenshots or error logs

## Additional Context
Any other relevant information
```

### Issue Labels

- `bug` - Something isn't working
- `feature` - Feature request
- `documentation` - Docs need improvement
- `good first issue` - Good for newcomers
- `help wanted` - Extra attention needed
- `in progress` - Someone is working on it

## Project Structure

```
observability-copilot/
├── backend/                       # Go backend
│   ├── cmd/server/main.go        # API & routes
│   ├── pkg/
│   │   ├── scanner/              # Repository scanning
│   │   ├── generator/            # Code generation
│   │   │   ├── generator.go      # Main dispatcher
│   │   │   ├── go_generator.go   # Go support
│   │   │   ├── python_generator.go
│   │   │   └── java_generator.go
│   │   ├── github/               # GitHub integration
│   │   ├── togglespec/           # Configuration
│   │   ├── models/               # Data models
│   │   └── db/                   # Database setup
│   ├── go.mod
│   ├── Dockerfile
│   └── k8s/                      # Kubernetes manifests
│
├── copilot-ui/                    # React frontend
│   ├── src/
│   │   ├── components/           # React components
│   │   │   ├── ImportForm.tsx    # Main component
│   │   │   ├── DashboardShell.tsx
│   │   │   └── ...
│   │   ├── config/api.ts         # API configuration
│   │   └── App.tsx
│   ├── package.json
│   ├── Dockerfile
│   └── public/
│
├── docs/                          # Documentation
├── k8s/                          # K8s manifests
├── config/                       # Config files
├── README.md                     # Main documentation
├── CONTRIBUTING.md              # This file
├── LICENSE                       # MIT License
└── .gitignore
```

## Testing

### Backend Tests

```bash
cd backend

# Run all tests
go test ./...

# Run specific package
go test ./pkg/scanner

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Frontend Tests

```bash
cd copilot-ui

# Run tests
npm test

# Run with coverage
npm test -- --coverage

# Run specific test file
npm test ImportForm
```

### Writing Tests

**Go test example:**
```go
func TestScanRepo(t *testing.T) {
    result, err := ScanRepo("https://github.com/user/repo.git", "repo-id")
    
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    
    if result.Framework != "Go" {
        t.Errorf("Expected framework Go, got %s", result.Framework)
    }
}
```

**React test example:**
```typescript
import { render, screen } from '@testing-library/react';
import ImportForm from './ImportForm';

test('renders import form', () => {
    render(<ImportForm />);
    expect(screen.getByText(/import/i)).toBeInTheDocument();
});
```

## Code Style

### Go

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Run `go fmt` before committing
- Keep lines under 100 characters
- Use meaningful variable names
- Add comments for exported functions

```go
// Good
func ScanRepo(url, id string) (*ScanResult, error) {
    // implementation
}

// Bad
func sr(u, i string) (*sr, error) {
    // implementation
}
```

### TypeScript/React

- Follow [Google TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html)
- Use functional components with hooks
- Add prop types (TypeScript)
- Keep components small and focused
- Use descriptive names

```typescript
// Good
interface ImportFormProps {
    onImportComplete?: (data: ImportResult) => void;
}

export function ImportForm({ onImportComplete }: ImportFormProps) {
    // component code
}

// Bad
export const IF = ({ oic }: any) => {
    // component code
}
```

### Comments

- Write comments that explain "why", not "what"
- Keep comments up-to-date
- Use TODO comments for future work

```go
// Good - explains why
// Use shallow clone for speed, full clone is unnecessary
cmd := exec.Command("git", "clone", "--depth=1", url, path)

// Bad - just repeats the code
// Clone the repository
cmd := exec.Command("git", "clone", "--depth=1", url, path)
```

## Documentation

### Updating README

- Keep main README focused and concise
- Add detailed docs to separate files
- Update examples when adding features
- Keep API documentation current
- Add screenshots for UI changes

### Adding New Features

Create documentation that includes:

1. **What** - What does the feature do?
2. **Why** - Why is it useful?
3. **How** - How to use it?
4. **Examples** - Real-world examples
5. **Limitations** - What doesn't it do?

### Code Comments

```go
// Package scanner provides repository analysis and framework detection.
package scanner

// ScanRepo scans a GitHub repository for framework type and existing
// instrumentation (Prometheus metrics, OpenTelemetry traces).
//
// The scan process:
// 1. Clones repository to temporary directory
// 2. Detects framework by checking for framework-specific files
// 3. Performs two-pass instrumentation detection
// 4. Cleans up temporary files
//
// Returns ScanResult with framework, services, and instrumentation status.
// Returns error if clone fails or framework cannot be determined.
func ScanRepo(repoURL, repoID string) (*ScanResult, error) {
```

## Questions?

- **Documentation** - Check README.md and docs/
- **Issues** - Open a GitHub issue
- **Discussions** - Use GitHub Discussions
- **Email** - Contact maintainers directly

## Recognition

Contributors will be recognized in:
- [CONTRIBUTORS.md](CONTRIBUTORS.md) file
- GitHub contributors page
- Release notes for major contributions

Thank you for contributing! 🎉

---

**Happy contributing! Let's make Observability Copilot amazing together! 🚀**
