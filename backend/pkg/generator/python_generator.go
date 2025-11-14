package generator

import "fmt"

func generatePythonInstrumentation(service, mode string) (*InstrumentationPlan, error) {
    plan := &InstrumentationPlan{
        Framework:   "Python",
        Service:     service,
        Mode:        mode,
        Description: fmt.Sprintf("Add OpenTelemetry instrumentation for %s (mode: %s)", service, mode),
    }

    // Add dependencies to requirements.txt
    if mode == "traces" || mode == "both" {
        plan.Changes = append(plan.Changes, FileChange{
            Path:   "requirements.txt",
            Action: "append",
            Content: `
# OpenTelemetry dependencies
opentelemetry-api>=1.20.0
opentelemetry-sdk>=1.20.0
opentelemetry-exporter-otlp-proto-grpc>=1.20.0
opentelemetry-instrumentation-flask>=0.41b0
opentelemetry-instrumentation-requests>=0.41b0`,
        })
    }

    if mode == "metrics" || mode == "both" {
        plan.Changes = append(plan.Changes, FileChange{
            Path:   "requirements.txt",
            Action: "append",
            Content: `
# Prometheus dependencies
prometheus-client>=0.19.0`,
        })
    }

    // Generate instrumentation code
    if mode == "traces" || mode == "both" {
        plan.Changes = append(plan.Changes, generatePythonTracer(service))
    }

    if mode == "metrics" || mode == "both" {
        plan.Changes = append(plan.Changes, generatePythonMetrics(service))
    }

    return plan, nil
}

func generatePythonTracer(service string) FileChange {
    code := fmt.Sprintf(`
# OpenTelemetry Tracer Initialization
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.instrumentation.flask import FlaskInstrumentor

def init_tracer():
    """Initialize OpenTelemetry tracer"""
    resource = Resource.create({"service.name": "%s"})
    
    tracer_provider = TracerProvider(resource=resource)
    
    # OTLP exporter
    otlp_exporter = OTLPSpanExporter(
        endpoint="http://otel-collector.observability.svc.cluster.local:4317",
        insecure=True
    )
    
    tracer_provider.add_span_processor(BatchSpanProcessor(otlp_exporter))
    trace.set_tracer_provider(tracer_provider)
    
    # Auto-instrument Flask (if using Flask)
    FlaskInstrumentor().instrument()
    
    print("✅ OpenTelemetry tracer initialized")

# Call this in your main app file before app.run()
# init_tracer()
`, service)

    return FileChange{
        Path:    "otel_config.py",
        Action:  "create",
        Content: code,
    }
}

func generatePythonMetrics(service string) FileChange {
    code := `
# Prometheus Metrics
from prometheus_client import Counter, Histogram, start_http_server, generate_latest
from flask import Response
import time

# Define metrics
http_requests_total = Counter(
    'http_requests_total',
    'Total HTTP requests',
    ['method', 'endpoint', 'status']
)

http_request_duration_seconds = Histogram(
    'http_request_duration_seconds',
    'HTTP request duration',
    ['method', 'endpoint']
)

def setup_metrics(app):
    """Setup Prometheus metrics for Flask app"""
    
    @app.before_request
    def before_request():
        request.start_time = time.time()
    
    @app.after_request
    def after_request(response):
        duration = time.time() - request.start_time
        http_requests_total.labels(
            method=request.method,
            endpoint=request.endpoint or 'unknown',
            status=response.status_code
        ).inc()
        
        http_request_duration_seconds.labels(
            method=request.method,
            endpoint=request.endpoint or 'unknown'
        ).observe(duration)
        
        return response
    
    @app.route('/metrics')
    def metrics():
        """Expose Prometheus metrics endpoint"""
        return Response(generate_latest(), mimetype='text/plain')
    
    print("✅ Prometheus metrics initialized")

# Call this in your main app file:
# setup_metrics(app)
`

    return FileChange{
        Path:    "metrics_config.py",
        Action:  "create",
        Content: code,
    }
}
