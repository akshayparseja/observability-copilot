
# OpenTelemetry Tracer Initialization
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.instrumentation.flask import FlaskInstrumentor

def init_tracer():
    """Initialize OpenTelemetry tracer"""
    resource = Resource.create({"service.name": "flask-app"})
    
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
