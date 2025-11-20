package scanner

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AnalyzePythonRepo performs source-level scanning of Python files to detect
// Flask app usage, Prometheus metrics, and OpenTelemetry instrumentation.
// Returns file-level candidates only when actual usage is detected (not just imports).
func AnalyzePythonRepo(root string) ([]Candidate, error) {
	log.Printf("[scanner][python] AnalyzePythonRepo started for %s", root)
	var metricFilesSet = map[string]struct{}{}
	var otelFilesSet = map[string]struct{}{}
	var flaskFilesSet = map[string]struct{}{}

	// Walk all .py files
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip vendor, venv, __pycache__, .git
			base := info.Name()
			if base == "venv" || base == "vendor" || base == "__pycache__" || base == ".git" || base == ".venv" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".py") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil // skip files we can't read
		}
		code := string(content)

		// Detect Flask app creation: Flask(__name__), Flask("...", ...)
		if strings.Contains(code, "Flask(__name__)") || strings.Contains(code, "Flask(") {
			if strings.Contains(code, "from flask import") || strings.Contains(code, "import flask") {
				flaskFilesSet[path] = struct{}{}
			}
		}

		// Detect Prometheus metrics usage (not just imports)
		// Look for actual usage: start_http_server, Counter, Histogram, generate_latest
		hasPromImport := strings.Contains(code, "from prometheus_client import") || strings.Contains(code, "import prometheus_client")
		if hasPromImport {
			// Check for actual usage patterns
			if strings.Contains(code, "start_http_server(") ||
				strings.Contains(code, "Counter(") ||
				strings.Contains(code, "Histogram(") ||
				strings.Contains(code, "Summary(") ||
				strings.Contains(code, "Gauge(") ||
				strings.Contains(code, "generate_latest(") ||
				strings.Contains(code, ".inc(") ||
				strings.Contains(code, ".observe(") {
				metricFilesSet[path] = struct{}{}
			}
		}

		// Detect OpenTelemetry instrumentation usage
		hasOtelImport := strings.Contains(code, "opentelemetry") || strings.Contains(code, "from opentelemetry")
		if hasOtelImport {
			// Check for actual tracer/instrumentation setup
			if strings.Contains(code, "TracerProvider(") ||
				strings.Contains(code, "FlaskInstrumentor") ||
				strings.Contains(code, "OTLPSpanExporter(") ||
				strings.Contains(code, "start_as_current_span(") ||
				strings.Contains(code, "set_tracer_provider(") ||
				strings.Contains(code, ".instrument()") {
				otelFilesSet[path] = struct{}{}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk failed: %w", err)
	}

	// Convert absolute paths to relative
	var metricFiles []string
	var otelFiles []string
	var flaskFiles []string

	for f := range metricFilesSet {
		metricFiles = append(metricFiles, strings.TrimPrefix(f, root+"/"))
	}
	for f := range otelFilesSet {
		otelFiles = append(otelFiles, strings.TrimPrefix(f, root+"/"))
	}
	for f := range flaskFilesSet {
		flaskFiles = append(flaskFiles, strings.TrimPrefix(f, root+"/"))
	}

	// Determine framework based on detected Flask usage
	framework := "Python"
	if len(flaskFiles) > 0 {
		framework = "Flask"
	} else {
		// Fallback: check requirements.txt for framework hints if no source-level detection
		framework = detectPythonFrameworkFromRequirements(root)
	}

	var candidates []Candidate
	if len(metricFiles) > 0 {
		candidates = append(candidates, Candidate{
			Language:    "Python",
			Framework:   framework,
			Kind:        "metrics",
			Patterns:    []string{"prometheus_client", "start_http_server", "Counter", "Histogram"},
			Files:       metricFiles,
			ServiceName: "python-service",
		})
	}
	if len(otelFiles) > 0 {
		candidates = append(candidates, Candidate{
			Language:    "Python",
			Framework:   framework,
			Kind:        "otel",
			Patterns:    []string{"TracerProvider", "FlaskInstrumentor", "OTLPSpanExporter"},
			Files:       otelFiles,
			ServiceName: "python-service",
		})
	}

	log.Printf("[scanner][python] AnalyzePythonRepo complete for %s: metrics=%d otel=%d", root, len(metricFiles), len(otelFiles))
	return candidates, nil
}

// detectPythonFrameworkFromRequirements is a fallback for framework detection
// when no source-level Flask app is found but dependencies suggest a framework.
func detectPythonFrameworkFromRequirements(path string) string {
	content, err := os.ReadFile(filepath.Join(path, "requirements.txt"))
	if err != nil {
		return "Python"
	}
	contentStr := strings.ToLower(string(content))

	if strings.Contains(contentStr, "flask") {
		return "Flask"
	} else if strings.Contains(contentStr, "django") {
		return "Django"
	} else if strings.Contains(contentStr, "fastapi") {
		return "FastAPI"
	}
	return "Python"
}

// Helper: check if Python source uses a specific pattern (for validation/testing)
func findPythonPattern(repoPath, pattern string) []string {
	cmd := exec.Command("grep", "-ril", "--include=*.py", pattern, repoPath)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, l := range lines {
		if l != "" {
			files = append(files, l)
		}
	}
	return files
}
