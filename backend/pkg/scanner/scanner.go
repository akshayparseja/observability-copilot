package scanner

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FrameworkDetection represents a detected framework/language in the repo
type FrameworkDetection struct {
	Language    string `json:"language"`     // "Go", "Python", "Java", etc.
	Framework   string `json:"framework"`    // "Gin", "Flask", "Spring Boot", etc.
	HasMetrics  bool   `json:"has_metrics"`  // Already has Prometheus metrics
	HasOTel     bool   `json:"has_otel"`     // Already has OpenTelemetry
	ServiceName string `json:"service_name"` // e.g., "go-service", "python-service"
}

// Candidate describes file-level matches where instrumentation could be inserted
type Candidate struct {
	Language    string   `json:"language"`
	Framework   string   `json:"framework"`
	Kind        string   `json:"kind"` // "metrics" or "otel"
	Patterns    []string `json:"patterns"`
	Files       []string `json:"files"`
	ServiceName string   `json:"service_name"`
}

// ScanResult contains all detected frameworks
type ScanResult struct {
	Frameworks []FrameworkDetection `json:"frameworks"`
	Services   []string             `json:"services"` // Deprecated, use Frameworks instead
	Candidates []Candidate          `json:"candidates"`
}

// ScanRepo scans a repository and detects ALL languages/frameworks present
// ScanRepo scans a repository and detects ALL languages/frameworks present
// If branch is provided (non-empty), the specified branch will be cloned.
func ScanRepo(repoURL, repoID, branch string) (*ScanResult, error) {
	log.Printf("[scanner] ScanRepo started: repoID=%s url=%s branch=%s", repoID, repoURL, branch)
	clonePath := filepath.Join("/tmp", repoID)
	os.RemoveAll(clonePath)

	// Clone repo (optionally a specific branch)
	var cmd *exec.Cmd
	if branch != "" {
		cmd = exec.Command("git", "clone", "--depth=1", "--branch", branch, repoURL, clonePath)
	} else {
		cmd = exec.Command("git", "clone", "--depth=1", repoURL, clonePath)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to clone: %v output: %s", err, string(out))
	}
	defer os.RemoveAll(clonePath)

	result := &ScanResult{
		Frameworks: []FrameworkDetection{},
		Services:   []string{},
	}

	// Detect Go
	if detectGo(clonePath) {
		log.Printf("[scanner] Detecting Go in %s", repoID)
		fwName := detectGoFramework(clonePath)
		log.Printf("[scanner] Detected Go framework: %s", fwName)
		// Use AST-based analysis for Go to produce precise candidates
		goCandidates, err := AnalyzeGoRepo(clonePath)
		if err != nil {
			// Fallback to grep-based detection on error
			metricFiles := detectMetrics(clonePath, "Go")
			otelFiles := detectOTel(clonePath, "Go")
			framework := FrameworkDetection{
				Language:    "Go",
				Framework:   fwName,
				HasMetrics:  len(metricFiles) > 0,
				HasOTel:     len(otelFiles) > 0,
				ServiceName: "go-service",
			}
			result.Frameworks = append(result.Frameworks, framework)
			result.Services = append(result.Services, "go-service")
			if len(metricFiles) > 0 {
				result.Candidates = append(result.Candidates, Candidate{
					Language:    "Go",
					Framework:   fwName,
					Kind:        "metrics",
					Patterns:    []string{"http.Handle(\"/metrics\"", "promhttp.Handler()", "prometheus.MustRegister("},
					Files:       metricFiles,
					ServiceName: "go-service",
				})
			}
			if len(otelFiles) > 0 {
				result.Candidates = append(result.Candidates, Candidate{
					Language:    "Go",
					Framework:   fwName,
					Kind:        "otel",
					Patterns:    []string{"tracer.Start(", "sdktrace.NewTracerProvider(", "otlptrace"},
					Files:       otelFiles,
					ServiceName: "go-service",
				})
			}
		} else {
			// convert candidates into framework detection summary
			hasMetrics := false
			hasOtel := false
			for _, c := range goCandidates {
				if c.Kind == "metrics" && len(c.Files) > 0 {
					hasMetrics = true
				}
				if c.Kind == "otel" && len(c.Files) > 0 {
					hasOtel = true
				}
				// normalize file paths to be relative to repo
				for i := range c.Files {
					c.Files[i] = strings.TrimPrefix(c.Files[i], clonePath+"/")
				}
				result.Candidates = append(result.Candidates, c)
			}
			framework := FrameworkDetection{
				Language:    "Go",
				Framework:   fwName,
				HasMetrics:  hasMetrics,
				HasOTel:     hasOtel,
				ServiceName: "go-service",
			}
			log.Printf("[scanner] Creating FrameworkDetection: Language=%s, Framework=%s, HasMetrics=%v, HasOTel=%v", framework.Language, framework.Framework, framework.HasMetrics, framework.HasOTel)
			result.Frameworks = append(result.Frameworks, framework)
			result.Services = append(result.Services, "go-service")
		}
	}

	// Detect Python
	if detectPython(clonePath) {
		log.Printf("[scanner] Detecting Python in %s", repoID)
		// Use AST-based analysis for Python to produce precise candidates
		pythonCandidates, err := AnalyzePythonRepo(clonePath)
		if err != nil {
			// Fallback to grep-based detection on error
			fwName := detectPythonFramework(clonePath)
			metricFiles := detectMetrics(clonePath, "Python")
			otelFiles := detectOTel(clonePath, "Python")
			framework := FrameworkDetection{
				Language:    "Python",
				Framework:   fwName,
				HasMetrics:  len(metricFiles) > 0,
				HasOTel:     len(otelFiles) > 0,
				ServiceName: "python-service",
			}
			log.Printf("[scanner] Detected Python framework: %s, metrics=%d, otel=%d", fwName, len(metricFiles), len(otelFiles))
			result.Frameworks = append(result.Frameworks, framework)
			result.Services = append(result.Services, "python-service")
			if len(metricFiles) > 0 {
				result.Candidates = append(result.Candidates, Candidate{
					Language:    "Python",
					Framework:   fwName,
					Kind:        "metrics",
					Patterns:    []string{"start_http_server(", "generate_latest(", "prometheus_client"},
					Files:       metricFiles,
					ServiceName: "python-service",
				})
			}
			if len(otelFiles) > 0 {
				result.Candidates = append(result.Candidates, Candidate{
					Language:    "Python",
					Framework:   fwName,
					Kind:        "otel",
					Patterns:    []string{"tracer.start_as_current_span(", "OTLPSpanExporter("},
					Files:       otelFiles,
					ServiceName: "python-service",
				})
			}
		} else {
			// Use AST-based candidates
			hasMetrics := false
			hasOtel := false
			fwName := "Python"
			for _, c := range pythonCandidates {
				if c.Kind == "metrics" && len(c.Files) > 0 {
					hasMetrics = true
				}
				if c.Kind == "otel" && len(c.Files) > 0 {
					hasOtel = true
				}
				if c.Framework != "Python" {
					fwName = c.Framework
				}
				// normalize file paths to be relative to repo
				for i := range c.Files {
					c.Files[i] = strings.TrimPrefix(c.Files[i], clonePath+"/")
				}
				result.Candidates = append(result.Candidates, c)
			}
			framework := FrameworkDetection{
				Language:    "Python",
				Framework:   fwName,
				HasMetrics:  hasMetrics,
				HasOTel:     hasOtel,
				ServiceName: "python-service",
			}
			log.Printf("[scanner] Python analysis complete: framework=%s, hasMetrics=%v, hasOtel=%v", fwName, hasMetrics, hasOtel)
			result.Frameworks = append(result.Frameworks, framework)
			result.Services = append(result.Services, "python-service")
		}
	}

	// Detect Java
	if detectJava(clonePath) {
		log.Printf("[scanner] Detecting Java in %s", repoID)
		fwName := detectJavaFramework(clonePath)
		metricFiles := detectMetrics(clonePath, "Java")
		otelFiles := detectOTel(clonePath, "Java")
		framework := FrameworkDetection{
			Language:    "Java",
			Framework:   fwName,
			HasMetrics:  len(metricFiles) > 0,
			HasOTel:     len(otelFiles) > 0,
			ServiceName: "java-service",
		}
		result.Frameworks = append(result.Frameworks, framework)
		result.Services = append(result.Services, "java-service")
		if len(metricFiles) > 0 {
			result.Candidates = append(result.Candidates, Candidate{
				Language:    "Java",
				Framework:   fwName,
				Kind:        "metrics",
				Patterns:    []string{"MeterRegistry", "PrometheusMeterRegistry"},
				Files:       metricFiles,
				ServiceName: "java-service",
			})
		}
		if len(otelFiles) > 0 {
			result.Candidates = append(result.Candidates, Candidate{
				Language:    "Java",
				Framework:   fwName,
				Kind:        "otel",
				Patterns:    []string{"tracer.spanBuilder(", "OtlpGrpcSpanExporter"},
				Files:       otelFiles,
				ServiceName: "java-service",
			})
		}
	}

	// Detect Node.js
	if detectNode(clonePath) {
		log.Printf("[scanner] Detecting Node.js in %s", repoID)
		fwName := detectNodeFramework(clonePath)
		metricFiles := detectMetrics(clonePath, "Node.js")
		otelFiles := detectOTel(clonePath, "Node.js")
		framework := FrameworkDetection{
			Language:    "Node.js",
			Framework:   fwName,
			HasMetrics:  len(metricFiles) > 0,
			HasOTel:     len(otelFiles) > 0,
			ServiceName: "nodejs-service",
		}
		result.Frameworks = append(result.Frameworks, framework)
		result.Services = append(result.Services, "nodejs-service")
		if len(metricFiles) > 0 {
			result.Candidates = append(result.Candidates, Candidate{
				Language:    "Node.js",
				Framework:   fwName,
				Kind:        "metrics",
				Patterns:    []string{"register.metrics()", "prom-client"},
				Files:       metricFiles,
				ServiceName: "nodejs-service",
			})
		}
		if len(otelFiles) > 0 {
			result.Candidates = append(result.Candidates, Candidate{
				Language:    "Node.js",
				Framework:   fwName,
				Kind:        "otel",
				Patterns:    []string{"tracer.startSpan(", "@opentelemetry/sdk-trace"},
				Files:       otelFiles,
				ServiceName: "nodejs-service",
			})
		}
	}

	// Detect .NET
	if detectDotnet(clonePath) {
		log.Printf("[scanner] Detecting .NET in %s", repoID)
		fwName := ".NET"
		metricFiles := detectMetrics(clonePath, ".NET")
		otelFiles := detectOTel(clonePath, ".NET")
		framework := FrameworkDetection{
			Language:    ".NET",
			Framework:   fwName,
			HasMetrics:  len(metricFiles) > 0,
			HasOTel:     len(otelFiles) > 0,
			ServiceName: "dotnet-service",
		}
		result.Frameworks = append(result.Frameworks, framework)
		result.Services = append(result.Services, "dotnet-service")
		if len(metricFiles) > 0 {
			result.Candidates = append(result.Candidates, Candidate{
				Language:    ".NET",
				Framework:   fwName,
				Kind:        "metrics",
				Patterns:    []string{"UsePrometheusServer"},
				Files:       metricFiles,
				ServiceName: "dotnet-service",
			})
		}
		if len(otelFiles) > 0 {
			result.Candidates = append(result.Candidates, Candidate{
				Language:    ".NET",
				Framework:   fwName,
				Kind:        "otel",
				Patterns:    []string{"ActivitySource"},
				Files:       otelFiles,
				ServiceName: "dotnet-service",
			})
		}
	}

	// Detect Rust
	if detectRust(clonePath) {
		log.Printf("[scanner] Detecting Rust in %s", repoID)
		fwName := "Rust"
		metricFiles := detectMetrics(clonePath, "Rust")
		otelFiles := detectOTel(clonePath, "Rust")
		framework := FrameworkDetection{
			Language:    "Rust",
			Framework:   fwName,
			HasMetrics:  len(metricFiles) > 0,
			HasOTel:     len(otelFiles) > 0,
			ServiceName: "rust-service",
		}
		result.Frameworks = append(result.Frameworks, framework)
		result.Services = append(result.Services, "rust-service")
		if len(metricFiles) > 0 {
			result.Candidates = append(result.Candidates, Candidate{
				Language:    "Rust",
				Framework:   fwName,
				Kind:        "metrics",
				Patterns:    []string{"prometheus::register"},
				Files:       metricFiles,
				ServiceName: "rust-service",
			})
		}
		if len(otelFiles) > 0 {
			result.Candidates = append(result.Candidates, Candidate{
				Language:    "Rust",
				Framework:   fwName,
				Kind:        "otel",
				Patterns:    []string{"opentelemetry::"},
				Files:       otelFiles,
				ServiceName: "rust-service",
			})
		}
	}

	log.Printf("[scanner] ScanRepo complete: repoID=%s frameworks=%d candidates=%d", repoID, len(result.Frameworks), len(result.Candidates))
	return result, nil
}

// Framework Detection Functions
func detectPython(path string) bool {
	// Detect Python by walking the repository looking for any .py files.
	found := false
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			base := filepath.Base(p)
			if base == "venv" || base == "vendor" || base == "__pycache__" || base == ".git" || base == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".py") {
			found = true
			return nil
		}
		return nil
	})
	return found
}

func detectPythonFramework(path string) string {
	// Try top-level requirements.txt first
	content, _ := os.ReadFile(filepath.Join(path, "requirements.txt"))
	contentStr := strings.ToLower(string(content))
	if strings.Contains(contentStr, "flask") {
		return "Flask"
	} else if strings.Contains(contentStr, "django") {
		return "Django"
	} else if strings.Contains(contentStr, "fastapi") {
		return "FastAPI"
	}

	// If not found, search for any requirements.txt in the tree and inspect it
	var foundFW string
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || foundFW != "" {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(p), "requirements.txt") {
			b, _ := os.ReadFile(p)
			s := strings.ToLower(string(b))
			if strings.Contains(s, "flask") {
				foundFW = "Flask"
			} else if strings.Contains(s, "django") {
				foundFW = "Django"
			} else if strings.Contains(s, "fastapi") {
				foundFW = "FastAPI"
			}
		}
		return nil
	})
	if foundFW != "" {
		return foundFW
	}
	return "Python"
}

func detectGo(path string) bool {
	// Search recursively for a go.mod file anywhere in the repo tree
	found := false
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			base := filepath.Base(p)
			if base == "vendor" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(p) == "go.mod" {
			found = true
			return nil
		}
		return nil
	})
	return found
}

func detectGoFramework(path string) string {
	// Look for the nearest go.mod in the tree and inspect it
	var modPath string
	_ = filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil || modPath != "" {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(p) == "go.mod" {
			modPath = p
		}
		return nil
	})
	if modPath != "" {
		content, _ := os.ReadFile(modPath)
		contentStr := string(content)
		if strings.Contains(contentStr, "github.com/gin-gonic/gin") {
			return "Gin"
		} else if strings.Contains(contentStr, "github.com/labstack/echo") {
			return "Echo"
		} else if strings.Contains(contentStr, "github.com/go-chi/chi") {
			return "Chi"
		} else if strings.Contains(contentStr, "github.com/gorilla/mux") {
			return "Gorilla Mux"
		}
	}
	return "Go"
}

func detectJava(path string) bool {
	files := []string{"pom.xml", "build.gradle", "build.gradle.kts"}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(path, f)); err == nil {
			return true
		}
	}
	return false
}

func detectJavaFramework(path string) string {
	pomContent, _ := os.ReadFile(filepath.Join(path, "pom.xml"))
	gradleContent, _ := os.ReadFile(filepath.Join(path, "build.gradle"))

	content := string(pomContent) + string(gradleContent)

	if strings.Contains(content, "spring-boot") {
		return "Spring Boot"
	} else if strings.Contains(content, "quarkus") {
		return "Quarkus"
	} else if strings.Contains(content, "micronaut") {
		return "Micronaut"
	}
	return "Java"
}

func detectDotnet(path string) bool {
	files, _ := filepath.Glob(filepath.Join(path, "*.csproj"))
	return len(files) > 0
}

func detectNode(path string) bool {
	_, err := os.Stat(filepath.Join(path, "package.json"))
	return err == nil
}

func detectNodeFramework(path string) string {
	content, _ := os.ReadFile(filepath.Join(path, "package.json"))
	contentStr := string(content)

	if strings.Contains(contentStr, "\"express\"") {
		return "Express"
	} else if strings.Contains(contentStr, "\"@nestjs/core\"") {
		return "NestJS"
	} else if strings.Contains(contentStr, "\"fastify\"") {
		return "Fastify"
	} else if strings.Contains(contentStr, "\"koa\"") {
		return "Koa"
	}
	return "Node.js"
}

func detectRust(path string) bool {
	_, err := os.Stat(filepath.Join(path, "Cargo.toml"))
	return err == nil
}

// Metrics Detection
// Returns a slice of file paths that matched any of the metrics patterns for the framework
func detectMetrics(path string, framework string) []string {
	metricsPatterns := map[string][]string{
		"Python": {
			"start_http_server(",
			"generate_latest(",
			"prometheus_client",
		},
		"Go": {
			"http.Handle(\"/metrics\"",
			"promhttp.Handler()",
			"prometheus.MustRegister(",
		},
		"Java": {
			"MeterRegistry",
			"PrometheusMeterRegistry",
		},
		".NET": {
			"UsePrometheusServer",
		},
		"Node.js": {
			"register.metrics()",
			"prom-client",
		},
		"Rust": {
			"prometheus::register",
		},
	}

	patterns := metricsPatterns[framework]
	if patterns == nil {
		return nil
	}

	filesSet := map[string]struct{}{}
	for _, pattern := range patterns {
		files := findFilesInRepo(path, pattern)
		for _, f := range files {
			filesSet[f] = struct{}{}
		}
	}

	var files []string
	for f := range filesSet {
		files = append(files, f)
	}
	return files
}

// OTel Detection
// Returns a slice of file paths that matched any of the otel patterns for the framework
func detectOTel(path string, framework string) []string {
	otelPatterns := map[string][]string{
		"Python": {
			"tracer.start_as_current_span(",
			"OTLPSpanExporter(",
		},
		"Go": {
			"tracer.Start(",
			"sdktrace.NewTracerProvider(",
			"otlptrace",
		},
		"Java": {
			"tracer.spanBuilder(",
			"OtlpGrpcSpanExporter",
		},
		".NET": {
			"ActivitySource",
		},
		"Node.js": {
			"tracer.startSpan(",
			"@opentelemetry/sdk-trace",
		},
		"Rust": {
			"opentelemetry::",
		},
	}

	patterns := otelPatterns[framework]
	if patterns == nil {
		return nil
	}

	filesSet := map[string]struct{}{}
	for _, pattern := range patterns {
		files := findFilesInRepo(path, pattern)
		for _, f := range files {
			filesSet[f] = struct{}{}
		}
	}

	var files []string
	for f := range filesSet {
		files = append(files, f)
	}
	return files
}

// Helper: Search pattern in repo
// Helper: run grep to find files containing the pattern (case-insensitive)
func findFilesInRepo(repoPath, pattern string) []string {
	cmd := exec.Command("grep", "-ril", pattern, repoPath)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// Only consider likely source files and ignore README/requirements/docs
	allowedExt := map[string]bool{
		".go":    true,
		".py":    true,
		".java":  true,
		".js":    true,
		".ts":    true,
		".cs":    true,
		".rs":    true,
		".kt":    true,
		".scala": true,
		".php":   true,
	}

	var files []string
	for _, l := range lines {
		if l == "" {
			continue
		}
		ext := filepath.Ext(l)
		if ext == "" {
			// no extension - skip (likely directories or scripts)
			continue
		}
		if !allowedExt[ext] {
			continue
		}

		// Open file and ensure the matched pattern appears in a non-commented line.
		b, err := os.ReadFile(l)
		if err != nil {
			continue
		}
		text := string(b)
		ok := false
		for _, line := range strings.Split(text, "\n") {
			if !strings.Contains(line, pattern) {
				continue
			}
			trimmed := strings.TrimSpace(line)
			// Language-agnostic comment checks for common single-line markers
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "<!--") {
				// commented line, skip
				continue
			}
			// also skip lines where pattern appears after // comment marker
			if idx := strings.Index(line, "//"); idx != -1 {
				if strings.Contains(line[:idx], pattern) {
					ok = true
					break
				}
				// pattern is in comment portion -> skip this occurrence
				continue
			}
			// crude check for block comment markers on the same line
			if strings.Contains(trimmed, "/*") || strings.Contains(trimmed, "*/") || strings.HasPrefix(trimmed, "*") {
				// skip this occurrence, better heuristics would parse code per-language
				continue
			}

			// If we reach here, the pattern appears on a non-commented line
			ok = true
			break
		}
		if ok {
			files = append(files, l)
		}
	}
	return files
}
