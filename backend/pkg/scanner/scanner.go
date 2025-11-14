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

// TWO-PASS METRICS DETECTION
// Pass 1: Check for registration/initialization
// Pass 2: Check for actual usage
func detectMetrics(path string, framework string) bool {
    // Registration patterns - metrics must be registered
    registrationPatterns := map[string][]string{
        "Python": {
            "prometheus_client.start_http_server(",
            "start_http_server(",
            "CollectorRegistry()",
        },
        "Go": {
            "prometheus.MustRegister(",
            "prometheus.Register(",
            "registry.MustRegister(",
            "registry.Register(",
        },
        "Java": {
            "new PrometheusMeterRegistry(",
            "new SimpleMeterRegistry(",
            "@Bean.*MeterRegistry",
        },
        ".NET": {
            "UsePrometheusServer(",
            "new KestrelMetricServer(",
        },
        "Node.js": {
            "register.registerMetric(",
            "collectDefaultMetrics(",
        },
        "Rust": {
            "prometheus::register(",
        },
    }

    // Usage patterns - metrics must be actually used
    usagePatterns := map[string][]string{
        "Python": {
            ".inc(",
            ".dec(",
            ".set(",
            ".observe(",
        },
        "Go": {
            ".Inc(",
            ".Dec(",
            ".Add(",
            ".Set(",
            ".Observe(",
            "promhttp.Handler()",
            "http.Handle(\"/metrics\"",
            "http.HandleFunc(\"/metrics\"",
            "router.GET(\"/metrics\"",
            "router.Handle(\"/metrics\"",
        },
        "Java": {
            ".counter(",
            ".gauge(",
            ".timer(",
            ".increment(",
        },
        ".NET": {
            ".Inc(",
            ".Set(",
            ".Observe(",
        },
        "Node.js": {
            ".inc(",
            ".set(",
            ".observe(",
            "register.metrics()",
        },
        "Rust": {
            ".inc(",
            ".set(",
            ".observe(",
        },
    }

    regPatterns := registrationPatterns[framework]
    usePatterns := usagePatterns[framework]
    
    if regPatterns == nil || usePatterns == nil {
        return false
    }

    // Must have BOTH registration AND usage
    hasRegistration := false
    hasUsage := false

    for _, pattern := range regPatterns {
        if searchInRepo(path, pattern) {
            hasRegistration = true
            break
        }
    }

    for _, pattern := range usePatterns {
        if searchInRepo(path, pattern) {
            hasUsage = true
            break
        }
    }

    return hasRegistration && hasUsage
}

// TWO-PASS OTEL DETECTION
// Pass 1: Check for tracer provider initialization
// Pass 2: Check for actual span creation/usage
func detectOTel(path string, framework string) bool {
    // Initialization patterns - tracer must be initialized
    initPatterns := map[string][]string{
        "Python": {
            "TracerProvider(",
            "OTLPSpanExporter(",
            "JaegerExporter(",
            "trace.set_tracer_provider(",
        },
        "Go": {
            "sdktrace.NewTracerProvider(",
            "otel.SetTracerProvider(",
            "otlptrace",
            "otlptracegrpc.New(",
            "jaeger.New(",
        },
        "Java": {
            "SdkTracerProvider.builder(",
            "OpenTelemetrySdk.builder(",
            "OtlpGrpcSpanExporter",
        },
        ".NET": {
            "TracerProvider.Default.GetTracer(",
            "new TracerProviderBuilder(",
        },
        "Node.js": {
            "new NodeTracerProvider(",
            "new BasicTracerProvider(",
            "new OTLPTraceExporter(",
        },
        "Rust": {
            "global::set_tracer_provider(",
            "opentelemetry::sdk::trace::TracerProvider",
        },
    }

    // Usage patterns - spans must be created
    usagePatterns := map[string][]string{
        "Python": {
            "tracer.start_as_current_span(",
            "tracer.start_span(",
            "@tracer.start_as_current_span",
        },
        "Go": {
            "tracer.Start(",
            "otel.Tracer(",
            "span.End(",
            "span.SetAttributes(",
        },
        "Java": {
            "tracer.spanBuilder(",
            "span.end(",
        },
        ".NET": {
            "tracer.StartActiveSpan(",
            "var span =",
        },
        "Node.js": {
            "tracer.startSpan(",
            "tracer.startActiveSpan(",
        },
        "Rust": {
            "tracer.in_span(",
            "tracer.start(",
        },
    }

    initPats := initPatterns[framework]
    usePats := usagePatterns[framework]
    
    if initPats == nil || usePats == nil {
        return false
    }

    // Must have BOTH initialization AND usage
    hasInit := false
    hasUsage := false

    for _, pattern := range initPats {
        if searchInRepo(path, pattern) {
            hasInit = true
            break
        }
    }

    for _, pattern := range usePats {
        if searchInRepo(path, pattern) {
            hasUsage = true
            break
        }
    }

    return hasInit && hasUsage
}

// Helper: Search pattern in repo files recursively
// Excludes vendor, node_modules, and test files to reduce false positives
func searchInRepo(repoPath, pattern string) bool {
    cmd := exec.Command("grep", "-r", "-i", 
        "--exclude-dir=vendor",
        "--exclude-dir=node_modules",
        "--exclude-dir=.git",
        "--exclude=*_test.go",
        "--exclude=*_test.py",
        "--exclude=*.test.js",
        pattern, repoPath)
    err := cmd.Run()
    return err == nil
}
