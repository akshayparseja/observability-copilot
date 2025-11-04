package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ScanResult struct {
	Framework   string   `json:"framework"`
	HasMetrics  bool     `json:"has_metrics"`
	HasOTel     bool     `json:"has_otel"`
	Services    []string `json:"services"`
}

func ScanRepo(repoURL, repoID string) (*ScanResult, error) {
	clonePath := filepath.Join("/tmp", repoID)
	os.RemoveAll(clonePath)

	cmd := exec.Command("git", "clone", "--depth=1", repoURL, clonePath)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to clone: %w", err)
	}

	result := &ScanResult{Services: []string{}}

	if detectPython(clonePath) {
		result.Framework = "Python"
		if detectDjango(clonePath) {
			result.Services = append(result.Services, "django-app")
		} else if detectFlask(clonePath) {
			result.Services = append(result.Services, "flask-app")
		}
	} else if detectGo(clonePath) {
		result.Framework = "Go"
		result.Services = append(result.Services, "go-service")
	} else if detectJava(clonePath) {
		result.Framework = "Java"
		result.Services = append(result.Services, "java-service")
	} else if detectDotnet(clonePath) {
		result.Framework = ".NET"
		result.Services = append(result.Services, "dotnet-service")
	} else if detectNode(clonePath) {
		result.Framework = "Node.js"
		result.Services = append(result.Services, "nodejs-service")
	} else if detectRust(clonePath) {
		result.Framework = "Rust"
		result.Services = append(result.Services, "rust-service")
	}

	result.HasMetrics = detectMetrics(clonePath, result.Framework)
	result.HasOTel = detectOTel(clonePath, result.Framework)

	os.RemoveAll(clonePath)
	return result, nil
}

// Framework Detection
func detectPython(path string) bool {
	files := []string{"requirements.txt", "setup.py", "pyproject.toml", "Pipfile"}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(path, f)); err == nil {
			return true
		}
	}
	return false
}

func detectDjango(path string) bool {
	content, _ := os.ReadFile(filepath.Join(path, "requirements.txt"))
	return strings.Contains(string(content), "django")
}

func detectFlask(path string) bool {
	content, _ := os.ReadFile(filepath.Join(path, "requirements.txt"))
	return strings.Contains(string(content), "flask")
}

func detectGo(path string) bool {
	_, err := os.Stat(filepath.Join(path, "go.mod"))
	return err == nil
}

func detectJava(path string) bool {
	files := []string{"pom.xml", "build.gradle"}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(path, f)); err == nil {
			return true
		}
	}
	return false
}

func detectDotnet(path string) bool {
	files, _ := filepath.Glob(filepath.Join(path, "*.csproj"))
	return len(files) > 0
}

func detectNode(path string) bool {
	_, err := os.Stat(filepath.Join(path, "package.json"))
	return err == nil
}

func detectRust(path string) bool {
	_, err := os.Stat(filepath.Join(path, "Cargo.toml"))
	return err == nil
}

// Metrics Detection - Check for ACTUAL EXPOSURE/REGISTRATION
func detectMetrics(path string, framework string) bool {
	metricsPatterns := map[string][]string{
		"Python": {
			"start_http_server(",
			"generate_latest(",
			"prometheus_client.start_http_server",
			"CollectorRegistry",
		},
		"Go": {
			"http.Handle(\"/metrics\"",
			"http.HandleFunc(\"/metrics\"",
			"promhttp.Handler()",
			"prometheus.MustRegister(",
			"registry.Register(",
		},
		"Java": {
			"MeterRegistry",
			"PrometheusMeterRegistry",
			"SimpleMeterRegistry",
			"getMeterRegistry()",
		},
		".NET": {
			"Prometheus.Exporter",
			"UsePrometheusServer",
			".CollectMetrics",
		},
		"Node.js": {
			"register.metrics()",
			"new Gauge(",
			"new Counter(",
			"new Histogram(",
		},
		"Rust": {
			"prometheus::register_gauge",
			"prometheus::Counter::new",
			"prometheus::Histogram::new",
		},
	}

	patterns := metricsPatterns[framework]
	if patterns == nil {
		return false
	}

	for _, pattern := range patterns {
		if searchInRepo(path, pattern) {
			return true
		}
	}
	return false
}

// OTel Detection - Check for ACTUAL INSTRUMENTATION/EXPORT
func detectOTel(path string, framework string) bool {
	otelPatterns := map[string][]string{
		"Python": {
			"tracer.start_as_current_span(",
			"Tracer.start_as_current_span",
			"JaegerExporter(",
			"OTLPSpanExporter(",
			"set_span_in_context(",
		},
		"Go": {
			"tracer.Start(",
			"span.End()",
			"sdktrace.NewTracerProvider(",
			"otlptrace.NewClient(",
			"jaeger.New(",
			"zipkin.New(",
		},
		"Java": {
			"tracer.spanBuilder(",
			"SdkTracerProvider",
			"OtlpGrpcSpanExporter",
			"JaegerExporter",
		},
		".NET": {
			"ActivitySource",
			"ActivityListener",
			"var activity = new Activity(",
			"new ZipkinExporter(",
		},
		"Node.js": {
			"tracer.startSpan(",
			"context.with(",
			"new JaegerExporter(",
			"new OTLPTraceExporter(",
		},
		"Rust": {
			"tracer.in_span(",
			"tracing::span(",
			"JaegerExporter",
			"OtlpExporter",
		},
	}

	patterns := otelPatterns[framework]
	if patterns == nil {
		return false
	}

	for _, pattern := range patterns {
		if searchInRepo(path, pattern) {
			return true
		}
	}
	return false
}


// Helper: Search pattern in repo files recursively
func searchInRepo(repoPath, pattern string) bool {
	cmd := exec.Command("grep", "-r", "-i", pattern, repoPath)
	err := cmd.Run()
	return err == nil
}
