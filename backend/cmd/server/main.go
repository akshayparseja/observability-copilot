package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"observability-copilot/pkg/generator"
	"observability-copilot/pkg/github"
	"observability-copilot/pkg/scanner"
)

var db *sql.DB

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// InitDB initializes database tables if they don't exist
func InitDB() error {
	schema := `
	-- Create repos table
	CREATE TABLE IF NOT EXISTS repos (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		github_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create services table
	CREATE TABLE IF NOT EXISTS services (
		id VARCHAR(255) PRIMARY KEY,
		repo_id VARCHAR(255) NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
		name VARCHAR(255) NOT NULL,
		framework VARCHAR(255),
		has_metrics BOOLEAN DEFAULT FALSE,
		has_otel BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create togglespecs table
	CREATE TABLE IF NOT EXISTS togglespecs (
		id VARCHAR(255) PRIMARY KEY,
		service_id VARCHAR(255) NOT NULL REFERENCES services(id) ON DELETE CASCADE,
		environment VARCHAR(50) NOT NULL,
		telemetry_mode VARCHAR(50) DEFAULT 'both',
		spec TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_services_repo_id ON services(repo_id);
	CREATE INDEX IF NOT EXISTS idx_togglespecs_service_id ON togglespecs(service_id);
	CREATE INDEX IF NOT EXISTS idx_togglespecs_env ON togglespecs(environment);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	log.Println("✅ Database schema initialized successfully")
	return nil
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/copilot?sslmode=disable"
	}
	fmt.Printf("DB URL: %s\n", dbURL)

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping DB: %v", err)
	}
	fmt.Println("✅ Connected to Postgres")

	// Initialize database schema
	err = InitDB()
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	router := gin.Default()

	// Expose Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	fmt.Println("✅ Enabled CORS middleware")

	router.Use(CORSMiddleware())

	// Health Check
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// GET /api/v1/repos - List all imported repositories
	fmt.Println("✅ addded repos endpoint")

	router.GET("/api/v1/repos", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, name, github_url FROM repos ORDER BY created_at DESC")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		repos := []map[string]string{}
		for rows.Next() {
			var id, name, githubURL string
			if err := rows.Scan(&id, &name, &githubURL); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			repos = append(repos, map[string]string{
				"id":         id,
				"name":       name,
				"github_url": githubURL, // ADD THIS
			})
		}

		c.JSON(200, repos)
	})
	router.POST("/api/v1/repos/:repo_id/create-pr", func(c *gin.Context) {
		repoID := c.Param("repo_id")

		var req struct {
			TelemetryMode string `json:"telemetry_mode"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		// Get repo info
		var githubURL string
		err := db.QueryRow("SELECT github_url FROM repos WHERE id = $1", repoID).Scan(&githubURL)
		if err != nil {
			c.JSON(404, gin.H{"error": "Repo not found"})
			return
		}

		// Get service info (framework, existing instrumentation)
		var framework, serviceName string
		var hasMetrics, hasOtel bool
		err = db.QueryRow(`
        SELECT framework, name, has_metrics, has_otel 
        FROM services 
        WHERE repo_id = $1 
        LIMIT 1
    `, repoID).Scan(&framework, &serviceName, &hasMetrics, &hasOtel)

		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to get service info"})
			return
		}

		// Determine what to add based on existing instrumentation
		modeToAdd := req.TelemetryMode

		// Smart detection: only add what's missing
		if req.TelemetryMode == "both" {
			if hasMetrics && hasOtel {
				c.JSON(400, gin.H{"error": "Already has both metrics and traces"})
				return
			} else if hasMetrics && !hasOtel {
				modeToAdd = "traces" // Only add traces
			} else if !hasMetrics && hasOtel {
				modeToAdd = "metrics" // Only add metrics
			}
			// else: add both (neither exists)
		} else if req.TelemetryMode == "metrics" && hasMetrics {
			c.JSON(400, gin.H{"error": "Already has metrics"})
			return
		} else if req.TelemetryMode == "traces" && hasOtel {
			c.JSON(400, gin.H{"error": "Already has traces"})
			return
		}

		// Generate instrumentation plan
		plan, err := generator.Generate(framework, serviceName, modeToAdd)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Create PR
		prURL, err := github.CreateInstrumentationPR(githubURL, plan, hasMetrics, hasOtel)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to create PR: %v", err)})
			return
		}

		c.JSON(200, gin.H{
			"pr_url":  prURL,
			"message": "Pull request created successfully",
		})
	})
	router.GET("/api/v1/repos/:repo_id/instrumentation-plan", func(c *gin.Context) {
		repoID := c.Param("repo_id")

		// Get service info from DB
		var framework, serviceName, telemetryMode string
		err := db.QueryRow(`
        SELECT s.framework, s.name, t.telemetry_mode
        FROM services s
        JOIN togglespecs t ON s.id = t.service_id
        WHERE s.repo_id = $1
        LIMIT 1
    `, repoID).Scan(&framework, &serviceName, &telemetryMode)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Generate instrumentation plan
		plan, err := generator.Generate(framework, serviceName, telemetryMode)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, plan)
	})
	// POST /api/v1/imports - Import a new repository
	router.POST("/api/v1/imports", func(c *gin.Context) {
		var req struct {
			GitHubURL     string `json:"github_url"`
			TelemetryMode string `json:"telemetry_mode"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		allowedModes := map[string]bool{"metrics": true, "traces": true, "both": true, "none": true}
		if !allowedModes[req.TelemetryMode] {
			c.JSON(400, gin.H{"error": "Invalid telemetry_mode, allowed values: metrics, traces, both, none"})
			return
		}

		parts := strings.Split(req.GitHubURL, "/")
		repoID := parts[len(parts)-1]
		repoID = strings.TrimSuffix(repoID, ".git")

		result, err := scanner.ScanRepo(req.GitHubURL, repoID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		_, err = db.Exec(
			"INSERT INTO repos (id, name, github_url, created_at, updated_at) VALUES ($1, $2, $3, NOW(), NOW()) ON CONFLICT (id) DO NOTHING",
			repoID, repoID, req.GitHubURL,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		for _, svc := range result.Services {
			serviceID := fmt.Sprintf("%s-%s", repoID, svc)
			_, err = db.Exec(
				"INSERT INTO services (id, repo_id, name, framework, has_metrics, has_otel, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW()) ON CONFLICT (id) DO NOTHING",
				serviceID, repoID, svc, result.Framework, result.HasMetrics, result.HasOTel,
			)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			spec := GenerateToggleSpecYAML(svc, req.TelemetryMode)
			toggleID := fmt.Sprintf("%s-dev", serviceID)

			_, err = db.Exec(
				`INSERT INTO togglespecs (id, service_id, environment, telemetry_mode, spec, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
				ON CONFLICT (id) DO NOTHING`,
				toggleID, serviceID, "dev", req.TelemetryMode, spec,
			)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}

		c.JSON(200, gin.H{
			"message": "Scan complete",
			"repo_id": repoID,
			"result":  result,
		})
	})

	// GET /api/v1/repos/:repo_id/plan
	router.GET("/api/v1/repos/:repo_id/plan", func(c *gin.Context) {
		repoID := c.Param("repo_id")

		// Get GitHub URL
		var githubURL string
		db.QueryRow("SELECT github_url FROM repos WHERE id = $1", repoID).Scan(&githubURL)

		rows, err := db.Query(
			"SELECT name, framework, has_metrics, has_otel FROM services WHERE repo_id = $1",
			repoID,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		services := []map[string]interface{}{}
		for rows.Next() {
			var name, framework string
			var hasMetrics, hasOtel bool
			rows.Scan(&name, &framework, &hasMetrics, &hasOtel)
			services = append(services, map[string]interface{}{
				"name":        name,
				"framework":   framework,
				"has_metrics": hasMetrics,
				"has_otel":    hasOtel,
			})
		}

		c.JSON(200, gin.H{
			"repo_id":  repoID,
			"services": services,
		})
	})

	// GET /api/v1/repos/:repo_id/services/:svc/toggles/:env
	router.GET("/api/v1/repos/:repo_id/services/:svc/toggles/:env", func(c *gin.Context) {
		repoID := c.Param("repo_id")
		svc := c.Param("svc")
		environment := c.Param("env")
		serviceID := fmt.Sprintf("%s-%s", repoID, svc)

		var spec string
		var telemetryMode string

		err := db.QueryRow("SELECT spec, telemetry_mode FROM togglespecs WHERE service_id = $1 AND environment = $2", serviceID, environment).Scan(&spec, &telemetryMode)
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "ToggleSpec not found"})
			return
		} else if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"spec":           spec,
			"telemetry_mode": telemetryMode,
		})
	})

	// PUT /api/v1/repos/:repo_id/services/:svc/toggles/:env
	router.PUT("/api/v1/repos/:repo_id/services/:svc/toggles/:env", func(c *gin.Context) {
		repoID := c.Param("repo_id")
		svc := c.Param("svc")
		environment := c.Param("env")
		serviceID := fmt.Sprintf("%s-%s", repoID, svc)

		var body struct {
			TelemetryMode string `json:"telemetry_mode"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "Invalid JSON"})
			return
		}

		allowedModes := map[string]bool{"metrics": true, "traces": true, "both": true, "none": true}
		if !allowedModes[body.TelemetryMode] {
			c.JSON(400, gin.H{"error": "Invalid telemetry_mode, allowed values: metrics, traces, both, none"})
			return
		}

		spec := GenerateToggleSpecYAML(svc, body.TelemetryMode)
		toggleID := fmt.Sprintf("%s-%s", serviceID, environment)

		_, err := db.Exec(`
			INSERT INTO togglespecs (id, service_id, environment, telemetry_mode, spec, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
			ON CONFLICT (id) DO UPDATE SET
				telemetry_mode = EXCLUDED.telemetry_mode,
				spec = EXCLUDED.spec,
				updated_at = NOW()
		`, toggleID, serviceID, environment, body.TelemetryMode, spec)

		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"message": "ToggleSpec saved"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	fmt.Printf("🚀 Server on :%s\n", port)
	router.Run(":" + port)
}

// GenerateToggleSpecYAML generates the YAML ToggleSpec string based on telemetry_mode.
func GenerateToggleSpecYAML(serviceName, telemetryMode string) string {
	switch telemetryMode {
	case "metrics":
		return fmt.Sprintf(`# ToggleSpec for %s
telemetry_mode: metrics
metrics:
  enabled: true
tracing:
  enabled: false
`, serviceName)
	case "traces":
		return fmt.Sprintf(`# ToggleSpec for %s
telemetry_mode: traces
metrics:
  enabled: false
tracing:
  enabled: true
`, serviceName)
	case "both":
		return fmt.Sprintf(`# ToggleSpec for %s
telemetry_mode: both
metrics:
  enabled: true
tracing:
  enabled: true
`, serviceName)
	default:
		return fmt.Sprintf(`# ToggleSpec for %s
telemetry_mode: none
metrics:
  enabled: false
tracing:
  enabled: false
`, serviceName)
	}
}
