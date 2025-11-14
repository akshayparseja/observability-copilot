package generator

import (
    "fmt"
)

func generateGoInstrumentation(service, mode string) (*InstrumentationPlan, error) {
    plan := &InstrumentationPlan{
        Framework:   "Go",
        Service:     service,
        Mode:        mode,
        Description: fmt.Sprintf("Add OpenTelemetry instrumentation for %s (mode: %s)", service, mode),
    }

    // Always add OTel dependencies
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

    // Generate tracer initialization code
    if mode == "traces" || mode == "both" {
        plan.Changes = append(plan.Changes, generateGoTracerInit(service))
        plan.Changes = append(plan.Changes, generateGoMiddleware(service))
    }

    // Generate Prometheus metrics code
    if mode == "metrics" || mode == "both" {
        plan.Changes = append(plan.Changes, generateGoMetrics(service))
    }

    return plan, nil
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

func generateGoMetrics(service string) FileChange {
    code := `
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
}

// Add to main() after router creation:
// Expose Prometheus metrics endpoint
router.GET("/metrics", gin.WrapH(promhttp.Handler()))
`

    return FileChange{
        Path:      "main.go",
        Action:    "append",
        Content:   code,
        LineAfter: "import (",
    }
}
