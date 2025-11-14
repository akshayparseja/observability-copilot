# Observability Copilot 🚀

**Intelligent, AI-powered observability automation for your GitHub repositories**

Observability Copilot automatically detects your application's framework, analyzes existing instrumentation (Prometheus metrics and OpenTelemetry traces), and generates pull requests with production-ready observability code. Say goodbye to manual instrumentation—let the copilot handle it for you!

## 🎯 What It Does

1. **Scans** your GitHub repository to detect the framework and programming language
2. **Analyzes** existing observability implementations:
   - Prometheus metrics exposition
   - OpenTelemetry tracer initialization & span creation
3. **Generates** framework-specific instrumentation code:
   - Adds dependencies to build manifests
   - Creates tracer/metrics initialization code
   - Sets up exporters and configuration
4. **Creates** a pull request automatically with all the changes
5. **Manages** telemetry modes through an intuitive dashboard

## 🌟 Key Features

- ✅ **Framework Detection** - Automatically identifies Go, Python, Java, Node.js, .NET, and Rust projects
- ✅ **Smart Instrumentation Detection** - Two-pass analysis (registration + usage) for accurate detection
- ✅ **Code Generation** - Production-ready instrumentation for multiple frameworks
- ✅ **GitHub Integration** - Creates PRs with properly formatted commits
- ✅ **Flexible Telemetry Modes** - Choose between metrics, traces, both, or none
- ✅ **Database Persistence** - Stores scan results and configuration
- ✅ **Web Dashboard** - Beautiful React UI for managing repositories and settings
- ✅ **Docker & Kubernetes Ready** - Deploy locally or to cloud

## 📊 Supported Frameworks

| Framework | Status | Metrics | Traces | Supported Build Files |
|-----------|:------:|:-------:|:------:|----------------------|
| **Go** | ✅ Full | ✅ | ✅ | `go.mod`, `main.go` |
| **Python** | ✅ Full | ✅ | ✅ | `requirements.txt`, `setup.py` |
| **Java** | ✅ Full | ✅ | ✅ | `pom.xml`, `build.gradle` |
| **Node.js** | 🚧 Planned | - | - | `package.json` |
| **.NET** | 🚧 Planned | - | - | `*.csproj` |
| **Rust** | 🚧 Planned | - | - | `Cargo.toml` |

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Frontend (React)                        │
│  Dashboard UI with Repository Management & Telemetry Config │
│  (copilot-ui/) - Port 80 (via Nginx)                       │
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP/REST
                       ▼
┌─────────────────────────────────────────────────────────────┐
│               Backend API (Go + Gin)                        │
│           cmd/server/main.go - Port 8000                    │
│                                                             │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │   Scanner   │  │  Generator   │  │ GitHub PR Bot   │   │
│  │ Framework   │  │ Code Gen for │  │ Commit & Push   │   │
│  │ Detection   │  │ All langs    │  │ Create PR       │   │
│  └─────────────┘  └──────────────┘  └─────────────────┘   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │          ToggleSpec Manager                          │   │
│  │  (Telemetry Mode Configuration & Persistence)       │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────┬──────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┬────────────┐
        ▼              ▼              ▼            ▼
    ┌────────┐   ┌──────────┐  ┌────────────┐ ┌──────────┐
    │ GitHub │   │PostgreSQL│  │ Git CLI    │ │ Git SSH  │
    │ API    │   │ Database │  │ (clone)    │ │ (auth)   │
    └────────┘   └──────────┘  └────────────┘ └──────────┘
```

### Component Breakdown

**Scanner** (`pkg/scanner/scanner.go`)
- Clones the target repository
- Detects framework by checking for framework-specific files
- Performs two-pass instrumentation detection:
  - **Pass 1**: Looks for registration calls (e.g., `prometheus.MustRegister`, `sdktrace.NewTracerProvider`)
  - **Pass 2**: Searches for actual usage (e.g., `http.Handle("/metrics")`, `tracer.Start()`)
- Returns `ScanResult` with framework, services, and instrumentation status

**Generator** (`pkg/generator/`)
- `generator.go` - Main dispatcher
- `go_generator.go` - Go-specific instrumentation
- `python_generator.go` - Python-specific instrumentation
- `java_generator.go` - Java-specific instrumentation
- Generates `InstrumentationPlan` with file changes:
  - Dependency additions (go.mod, requirements.txt, pom.xml)
  - Tracer initialization code
  - Metrics setup code
  - Configuration files

**GitHub Integration** (`pkg/github/pr.go`)
- Clones repository to temporary directory
- Creates feature branch (`feat/add-prometheus-metrics`, `feat/add-opentelemetry-traces`, etc.)
- Applies generated code changes
- Commits with descriptive message
- Pushes to origin
- Creates PR via GitHub API

**ToggleSpec Manager** (`pkg/togglespec/`)
- Generates YAML-based telemetry configuration
- Supports four modes: `metrics`, `traces`, `both`, `none`
- Persists configuration to database for future reference

## 📋 Database Schema

The application uses PostgreSQL with three main tables:

### `repos` - Repository Metadata
```sql
CREATE TABLE repos (
  id VARCHAR(255) PRIMARY KEY,           -- Extracted from repo URL
  name VARCHAR(255) NOT NULL,            -- Repository name
  github_url TEXT NOT NULL,              -- Full GitHub URL
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `services` - Detected Services/Applications
```sql
CREATE TABLE services (
  id VARCHAR(255) PRIMARY KEY,           -- Format: "{repo_id}-{service_name}"
  repo_id VARCHAR(255) NOT NULL,         -- Foreign key to repos
  name VARCHAR(255) NOT NULL,            -- Service name (e.g., "go-service")
  framework VARCHAR(255),                -- Detected framework (Go, Python, Java, etc.)
  has_metrics BOOLEAN DEFAULT FALSE,     -- Prometheus metrics detected
  has_otel BOOLEAN DEFAULT FALSE,        -- OpenTelemetry traces detected
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `togglespecs` - Telemetry Configuration per Environment
```sql
CREATE TABLE togglespecs (
  id VARCHAR(255) PRIMARY KEY,           -- Format: "{service_id}-{environment}"
  service_id VARCHAR(255) NOT NULL,      -- Foreign key to services
  environment VARCHAR(50) NOT NULL,      -- Environment (dev, staging, prod, etc.)
  telemetry_mode VARCHAR(50) DEFAULT 'both',  -- Current mode: metrics|traces|both|none
  spec TEXT,                             -- YAML configuration
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## 🔌 REST API Endpoints

### Health & Status

```bash
GET /api/v1/health
# Response: { "status": "ok" }
```

### Repository Management

```bash
# List all imported repositories
GET /api/v1/repos
# Response: [{ id, name, github_url }, ...]

# Scan a repository and store results
POST /api/v1/imports
# Body: { "github_url": "https://github.com/user/repo.git", "telemetry_mode": "both" }
# Response: { "message": "Scan complete", "repo_id": "...", "result": {...} }

# Get instrumentation plan for repository
GET /api/v1/repos/:repo_id/plan
# Response: { "repo_id": "...", "services": [...], "github_url": "..." }
```

### Telemetry Configuration

```bash
# Get telemetry config for a service in an environment
GET /api/v1/repos/:repo_id/services/:svc/toggles/:env
# Response: { "spec": "...", "telemetry_mode": "both" }

# Update telemetry mode
PUT /api/v1/repos/:repo_id/services/:svc/toggles/:env
# Body: { "telemetry_mode": "metrics" }
# Response: { "message": "ToggleSpec saved" }
```

### Pull Request Creation

```bash
# Create instrumentation PR for repository
POST /api/v1/repos/:repo_id/create-pr
# Body: { "telemetry_mode": "both" }
# Response: { "pr_url": "https://github.com/user/repo/pull/123", "message": "..." }
```

## 🎮 Telemetry Modes

The platform supports four flexible telemetry modes:

| Mode | Metrics | Traces | Use Case |
|------|:-------:|:------:|----------|
| **metrics** | ✅ | ❌ | Cost-conscious, basic monitoring |
| **traces** | ❌ | ✅ | Debugging, performance analysis, error tracking |
| **both** | ✅ | ✅ | Complete observability (recommended) |
| **none** | ❌ | ❌ | Disable telemetry collection |

### Smart Mode Selection

The UI automatically determines available modes based on existing instrumentation:

```
No metrics + No traces → All modes available (suggest "both")
Has metrics + No traces → Only "traces" and "none" available
No metrics + Has traces → Only "metrics" and "none" available
Has metrics + Has traces → Only "none" available
```

This prevents duplicate instrumentation and keeps code clean.

## 🚀 Getting Started

### Prerequisites

- **Docker** & **Docker Compose** (recommended for local dev)
- **Git** (for cloning repositories)
- **PostgreSQL 13+** (or use Docker)
- **Go 1.21+** (for backend development only)
- **Node.js 18+** (for frontend development only)
- **GitHub Token** (for PR creation) - [Get one here](https://github.com/settings/tokens)

### Option 1: Local Development (Backend + Frontend Separately)

**1. Start PostgreSQL**
```bash
docker run -d \
  --name observability-copilot-db \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=copilot \
  -p 5432:5432 \
  postgres:13
```

**2. Run Backend Server**
```bash
cd backend

export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"  # Your GitHub token
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/copilot?sslmode=disable"
export PORT=8000

go mod download
go mod tidy
go run ./cmd/server/main.go
```

Backend will start on `http://localhost:8000`

**3. Run Frontend**
```bash
cd copilot-ui

npm install
npm start
```

Frontend will start on `http://localhost:3000` with automatic proxy to backend

### Option 2: Kubernetes Deployment

**1. Create namespace**
```bash
kubectl apply -f k8s/namespaces/observability-ns.yaml
```

**2. Create ConfigMap for backend**
```bash
kubectl create configmap backend-config \
  -n observability-copilot \
  --from-literal=DATABASE_URL="postgres://postgres:postgres@postgres:5432/copilot?sslmode=disable"
```

**3. Deploy database**
```bash
# Helm (recommended)
helm install postgres bitnami/postgresql \
  -n observability-copilot \
  --set auth.password=postgres

# Or using manifests
bash backend/k8s/db/startup.sh
```

**4. Build and push images**
```bash
# Backend
docker build -t your-registry/observability-copilot-backend:latest ./backend
docker push your-registry/observability-copilot-backend:latest

# Frontend
cd copilot-ui && npm run build && cd ..
docker build -t your-registry/observability-copilot-frontend:latest ./copilot-ui
docker push your-registry/observability-copilot-frontend:latest
```

**5. Deploy services**
```bash
# Update image references in k8s manifests
kubectl apply -f backend/k8s/backend/deployment.yaml
kubectl apply -f backend/k8s/backend/service.yaml
kubectl apply -f k8s/ingress.yaml
```

**6. Access the dashboard**
```bash
kubectl port-forward svc/frontend 3000:80 -n observability-copilot
# Then visit http://localhost:3000
```

## 📖 How It Works - Step by Step

### 1. Scan a Repository

User enters GitHub URL in the dashboard (e.g., `https://github.com/user/my-go-app.git`)

```
User Input → POST /api/v1/imports → Scanner → Database
```

The scanner:
1. Creates temp dir in `/tmp/{repo_id}`
2. Runs: `git clone --depth=1 <url> <path>` (shallow clone for speed)
3. Detects framework by checking for files:
   - Go: `go.mod`
   - Python: `requirements.txt`, `setup.py`, `pyproject.toml`, `Pipfile`
   - Java: `pom.xml`, `build.gradle`
   - etc.
4. Uses `grep -r -i` to search for instrumentation patterns:
   - **Metrics patterns**: `prometheus.MustRegister`, `http.Handle("/metrics")`, etc.
   - **Trace patterns**: `tracer.Start`, `sdktrace.NewTracerProvider`, `OTLPSpanExporter`, etc.
5. Returns `ScanResult` with framework and instrumentation status
6. Stores repo and services in database

### 2. Review Results & Select Mode

Frontend displays scan results with smart mode recommendations:

```
Framework: Go
Has Metrics: ❌ No
Has Traces: ❌ No

Suggested Mode: "both" ✨

Available Options:
  ○ Metrics (Prometheus)
  ○ Traces (OpenTelemetry)
  ● Both (Hybrid) ← Default
  ○ None
```

User can override the suggestion if needed.

### 3. Generate & Create PR

User clicks "Create Pull Request" button

```
Selection → POST /api/v1/repos/:id/create-pr → Generator → GitHub
```

The backend:
1. Fetches repo info and existing instrumentation status
2. Calls `Generate(framework, service, mode)` to create plan
3. Calls `CreateInstrumentationPR(url, plan, hasMetrics, hasOtel)` which:
   - Clones repo to temp dir
   - Creates feature branch: `feat/add-prometheus-metrics` or `feat/add-opentelemetry-traces`
   - For each file in plan:
     - If `action: "append"` → add content to end of file
     - If `action: "modify"` → find line anchor and insert after
   - Git commits: `"chore: add observability instrumentation"`
   - Git pushes to origin
   - Creates PR via GitHub API with description of changes
4. Returns PR URL to frontend

User sees: "✅ PR created! [Open PR](https://github.com/...)" and can click to review

### 4. Manage Configuration

Once repository is scanned, user can:

- View current telemetry mode in dashboard
- Switch between available modes (respecting smart detection)
- Update environment-specific settings
- View generated ToggleSpec YAML

All changes are persisted to the database.

## 📝 Code Generation Examples

### Go Service - "both" mode

**Changes applied to `go.mod`:**
```go
require (
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/sdk v1.21.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
    go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.46.1
    github.com/prometheus/client_golang v1.17.0
)
```

**Changes to `main.go`:**
- Add tracer initialization function
- Add prometheus metrics setup
- Add Gin middleware for automatic tracing
- Add `/metrics` endpoint exposure

### Python Service - "metrics" mode

**Changes to `requirements.txt`:**
```
prometheus-client==0.18.0
```

**Changes to `main.py` or similar:**
- Import prometheus client
- Create metrics (counters, gauges, histograms)
- Add metrics endpoint
- Create request middleware to track metrics

### Java Service - "traces" mode

**Changes to `pom.xml`:**
```xml
<dependency>
    <groupId>io.opentelemetry</groupId>
    <artifactId>opentelemetry-sdk</artifactId>
    <version>1.21.0</version>
</dependency>
```

**Changes to application code:**
- Initialize OTel tracer provider
- Configure OTLP exporter
- Add tracer to HTTP client and servlet filters

## 🔐 Security Considerations

### GitHub Token Management

- Store `GITHUB_TOKEN` as a secret (never commit)
- Set as environment variable before starting backend
- Token requires `repo` scope (read/write) for PR creation
- Consider using GitHub App for production deployments

### Repository Cloning

**Current implementation:**
- Uses shallow clone (`--depth=1`) for speed
- Clones to `/tmp/{repo_id}` (world-readable by default)
- Removes cloned directory after processing

**Security notes:**
- Only clones public repositories (requires token for private repos)
- No validation of repository source (use GITHUB_TOKEN with limited org access)
- Consider adding repository URL whitelist in production
- Add timeout to clone operation to prevent hanging on large repos

### Database

- PostgreSQL credentials should be in environment variables
- Use strong passwords in production
- Consider enabling SSL for database connections
- Regularly backup the database

### API

- All endpoints require CORS headers (enabled in backend)
- No authentication currently implemented (add before production use)
- Rate limit GitHub API calls in production

## 🛠️ Development

### Building Images

**Backend:**
```bash
cd backend
docker build -t observability-copilot-backend:latest .
```

**Frontend:**
```bash
cd copilot-ui
npm run build
docker build -t observability-copilot-frontend:latest .
```

### Key Dependencies

**Backend (Go 1.21)**
- `github.com/gin-gonic/gin` - HTTP framework
- `github.com/lib/pq` - PostgreSQL driver

**Frontend (React 19 + TypeScript)**
- `@mui/material` - UI component library
- `axios` - HTTP client
- `react-scripts` - Build tooling

## 📚 API Usage Examples

### 1. Scan a Repository

```bash
curl -X POST http://localhost:8000/api/v1/imports \
  -H "Content-Type: application/json" \
  -d '{
    "github_url": "https://github.com/kubernetes/kubernetes.git",
    "telemetry_mode": "both"
  }'
```

**Response:**
```json
{
  "message": "Scan complete",
  "repo_id": "kubernetes",
  "result": {
    "framework": "Go",
    "services": ["kubernetes-service"],
    "has_metrics": true,
    "has_otel": false
  }
}
```

### 2. Get Instrumentation Plan

```bash
curl http://localhost:8000/api/v1/repos/kubernetes/plan
```

**Response:**
```json
{
  "repo_id": "kubernetes",
  "services": [
    {
      "name": "kubernetes-service",
      "framework": "Go",
      "has_metrics": true,
      "has_otel": false
    }
  ]
}
```

### 3. Create Pull Request

```bash
curl -X POST http://localhost:8000/api/v1/repos/kubernetes/create-pr \
  -H "Content-Type: application/json" \
  -d '{ "telemetry_mode": "both" }'
```

**Response:**
```json
{
  "pr_url": "https://github.com/kubernetes/kubernetes/pull/120456",
  "message": "Pull request created successfully"
}
```

### 4. Get/Update Telemetry Config

```bash
# Get current config
curl http://localhost:8000/api/v1/repos/kubernetes/services/kubernetes-service/toggles/dev

# Update config
curl -X PUT http://localhost:8000/api/v1/repos/kubernetes/services/kubernetes-service/toggles/dev \
  -H "Content-Type: application/json" \
  -d '{ "telemetry_mode": "metrics" }'
```

## 🐛 Troubleshooting

### Backend Issues

**Problem: `Failed to connect to database`**
- Check PostgreSQL is running: `docker ps | grep postgres`
- Verify DATABASE_URL: `echo $DATABASE_URL`
- Check credentials are correct

**Solution:**
```bash
docker run -d \
  --name copilot-db \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=copilot \
  -p 5432:5432 \
  postgres:13
```

**Problem: `GITHUB_TOKEN not set`**
- Backend cannot create PRs without token
- Generate one: https://github.com/settings/tokens (scope: `repo`)

**Solution:**
```bash
export GITHUB_TOKEN="ghp_xxxxxxxxxxxx"
go run ./cmd/server/main.go
```


### Frontend Issues

**Problem: `API requests return 404`**
- Backend URL is incorrect
- Check `src/config/api.ts` has correct base URL
- In Docker, use `http://backend:8000/api`

**Problem: `CORS errors`**
- Backend CORS middleware not enabled
- Check backend is running with CORSMiddleware() applied

**Solution:** Verify backend logs show "✅ Enabled CORS middleware"

### Repository Scanning Issues

**Problem: `Framework detection returned empty`**
- Repository doesn't have expected framework files
- Ensure `go.mod`, `package.json`, `pom.xml`, etc. exist

**Problem: `git clone timeout`**
- Large repository or slow network
- Currently no timeout implemented (TODO: add timeout)

**Solution:** Try with smaller public repo first, or check network

## 📋 Roadmap

### Phase 1 (Current)
- ✅ Go, Python, Java support
- ✅ Metrics & Traces detection
- ✅ PR creation automation
- ✅ React dashboard
- ✅ ToggleSpec configuration

### Phase 2 (In Progress)
- 🚧 Node.js instrumentation
- 🚧 .NET instrumentation
- 🚧 Rust instrumentation

### Phase 3 ( To be Planned)


## 🤝 Contributing

We welcome contributions! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines on:

- Setting up your development environment
- Code style and conventions
- Commit message format
- Pull request process
- Testing requirements
- Documentation standards

### Quick Start for Contributors

```bash
# Fork and clone
git clone https://github.com/YOUR_USERNAME/observability-copilot.git
cd observability-copilot

# Create feature branch
git checkout -b feat/your-feature-name

# Make changes, test, and commit
git commit -m "feat(scope): description of changes"

# Push and create PR
git push origin feat/your-feature-name
```

### Areas to Help

1. **Add Framework Support** - Extend `pkg/generator/` with Node.js, .NET, Rust
2. **Improve Detection** - Enhance pattern matching in `pkg/scanner/`
3. **Add Authentication** - Implement user auth and API keys
4. **UI Improvements** - Update React components for better UX
5. **Testing** - Add comprehensive unit and integration tests
6. **Security** - Review and improve security implementations
7. **Documentation** - Add examples, guides, and troubleshooting

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed contribution guidelines and areas.

## 📄 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

### What This Means

You are free to:
- ✅ Use this project commercially
- ✅ Modify and distribute the code
- ✅ Use for private projects
- ✅ Include in proprietary software

All we ask is that you:
- 📋 Include the original copyright notice
- 📋 Include the license text

**More info:** [MIT License on OpenSource.org](https://opensource.org/licenses/MIT)

## 💬 Support

- 📧 **Issues** - Open an issue on GitHub for bugs and feature requests
- 📖 **Documentation** - Check [`docs/OBSERVABILITY_SETUP.md`](docs/OBSERVABILITY_SETUP.md) for detailed observability stack information
- 💡 **Ideas** - Discussions welcome in GitHub Discussions

## 🙏 Acknowledgments

Built with modern open-source technologies:

- **Go** - Fast, concurrent backend
- **React 19** - Modern frontend framework
- **PostgreSQL** - Reliable data persistence
- **OpenTelemetry** - Industry-standard distributed tracing
- **Prometheus** - Battle-tested metrics collection
- **Gin** - High-performance web framework
- **Material-UI** - Beautiful component library

## 📞 Quick Links

- [GitHub Repository](https://github.com/akshayparseja/observability-copilot)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [GitHub API Reference](https://docs.github.com/en/rest)
- [Create a GitHub Personal Access Token](https://github.com/settings/tokens)

---

**Made with ❤️ by the Observability Copilot Team**

*Automating observability, one repository at a time.*
