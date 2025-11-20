package generator

import (
	"fmt"
	"observability-copilot/pkg/scanner"
	"strings"
)

func generateGoInstrumentation(service, mode string, candidates []scanner.Candidate) (*InstrumentationPlan, error) {
	plan := &InstrumentationPlan{
		Framework:   "Go",
		Service:     service,
		Mode:        mode,
		Description: fmt.Sprintf("Add Prometheus/OpenTelemetry instrumentation for %s (mode: %s)", service, mode),
	}

	// If traces requested, still add OTel dependencies
	if mode == "traces" || mode == "both" {
		plan.Changes = append(plan.Changes, FileChange{
			Path:   "go.mod",
			Action: "append",
			Content: `
require (
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/sdk v1.21.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.21.0
    go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.46.1
)`,
		})
		plan.Changes = append(plan.Changes, generateGoTracerInit(service))
		plan.Changes = append(plan.Changes, generateGoMiddleware(service))
	}

	// For metrics: create a centralized metrics file and add /metrics endpoint in router
	if mode == "metrics" || mode == "both" {
		// Create metrics/metrics.go if not exists
		metricsContent := generateMetricsFileContent(service)
		plan.Changes = append(plan.Changes, FileChange{
			Path:    "metrics/metrics.go",
			Action:  "create",
			Content: metricsContent,
		})

		// Determine insertion point for router metrics endpoint
		targetFile := "main.go"
		if candidates != nil {
			for _, c := range candidates {
				if c.Kind == "metrics" && len(c.Files) > 0 {
					// pick first matched file relative path
					targetFile = c.Files[0]
					break
				}
			}
		}

		// Add import for promhttp and prometheus if necessary (insert after import()
		// only add promhttp import to avoid introducing unused prometheus import in main packages
		importSnippet := `
	"github.com/prometheus/client_golang/prometheus/promhttp"
`
		plan.Changes = append(plan.Changes, FileChange{
			Path:      targetFile,
			Action:    "append",
			Content:   importSnippet,
			LineAfter: "import (",
		})

		// Add router endpoint after router initialization
		routerSnippet := `
// Expose Prometheus metrics endpoint
router.GET("/metrics", gin.WrapH(promhttp.Handler()))
`
		plan.Changes = append(plan.Changes, FileChange{
			Path:      targetFile,
			Action:    "modify",
			Content:   routerSnippet,
			LineAfter: "router := gin.Default()",
		})
	}

	return plan, nil
}

func generateMetricsFileContent(service string) string {
	return fmt.Sprintf(`package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    HttpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "%s_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    HttpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "%s_http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
)

func init() {
    prometheus.MustRegister(HttpRequestsTotal)
    prometheus.MustRegister(HttpRequestDuration)
}
`, strings.ReplaceAll(service, "-", "_"), strings.ReplaceAll(service, "-", "_"))
}

func generateGoTracerInit(service string) FileChange {
	code := fmt.Sprintf(`
import (
    "context"
    "log"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// initTracer initializes the OpenTelemetry tracer
func initTracer() (*sdktrace.TracerProvider, error) {
    ctx := context.Background()
    
    // Create OTLP exporter
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint("otel-collector.observability.svc.cluster.local:4317"),
        otlptracegrpc.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }

    // Create tracer provider
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("%s"),
        )),
    )

    otel.SetTracerProvider(tp)
    log.Println("✅ OpenTelemetry tracer initialized")
    return tp, nil
}`, service)

	return FileChange{
		Path:      "main.go",
		Action:    "append",
		Content:   code,
		LineAfter: "import (",
	}
}

func generateGoMiddleware(service string) FileChange {
	code := fmt.Sprintf(`
import "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

// Add to main() function after router creation:
// Initialize tracer
tp, err := initTracer()
if err != nil {
    log.Fatalf("Failed to initialize tracer: %%v", err)
}
defer func() {
    if err := tp.Shutdown(context.Background()); err != nil {
        log.Printf("Error shutting down tracer: %%v", err)
    }
}()

// Add OTel middleware to Gin router
router.Use(otelgin.Middleware("%s"))
`, service)

	return FileChange{
		Path:      "main.go",
		Action:    "modify",
		Content:   code,
		LineAfter: "router := gin.Default()",
	}
}
