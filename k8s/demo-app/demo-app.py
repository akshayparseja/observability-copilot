import sys
import os
import logging

# Enable OTel debug logging
logging.basicConfig(level=logging.DEBUG)
os.environ['OTEL_LOG_LEVEL'] = 'debug'

print("=" * 80, flush=True)
print("STARTING DEMO-APP", flush=True)
print("=" * 80, flush=True)

from flask import Flask
from prometheus_client import Counter, generate_latest

app = Flask(__name__)
http_requests_total = Counter('http_requests_total', 'Total HTTP requests', ['method', 'endpoint', 'status'])

print("[1] Flask app created", flush=True)

# OTel setup with detailed logging
try:
    print("[2] Starting OTel imports...", flush=True)
    
    from opentelemetry import trace
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.sdk.trace.export import BatchSpanProcessor
    from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
    from opentelemetry.instrumentation.flask import FlaskInstrumentor
    from opentelemetry.instrumentation.requests import RequestsInstrumentor
    from flask import Flask, Response  # ← ADD Response HERE
    from prometheus_client import Counter, generate_latest, CONTENT_TYPE_LATEST  
    print("[3] OTel imports OK", flush=True)
    print("[4] Creating OTLP exporter to otel-collector.observability.svc.cluster.local:4317", flush=True)
    
    exporter = OTLPSpanExporter(
        endpoint="otel-collector.observability.svc.cluster.local:4317", 
        insecure=True
    )
    print("[5] Exporter created", flush=True)
    
    tracer_provider = TracerProvider()
    print("[6] TracerProvider created", flush=True)
    
    tracer_provider.add_span_processor(BatchSpanProcessor(exporter))
    print("[7] SpanProcessor added", flush=True)
    
    trace.set_tracer_provider(tracer_provider)
    print("[8] Global tracer provider set", flush=True)
    
    FlaskInstrumentor().instrument_app(app)
    print("[9] Flask instrumented", flush=True)
    
    RequestsInstrumentor().instrument()
    print("[10] Requests instrumented", flush=True)
    
    print("[SUCCESS] ✓✓✓ OTel FULLY ACTIVE - Traces will be sent!", flush=True)
    
except Exception as e:
    print(f"[ERROR] OTel initialization FAILED at step: {e}", flush=True)
    import traceback
    traceback.print_exc()

@app.route('/')
def hello():
    http_requests_total.labels(method='GET', endpoint='/', status=200).inc()
    return 'Hello from demo-app!', 200

@app.route('/health')
def health():
    return 'OK', 200

@app.route('/metrics')
def metrics():
    return Response(generate_latest(), mimetype=CONTENT_TYPE_LATEST)

if __name__ == '__main__':
    print("[RUN] Starting Flask server on 0.0.0.0:8080", flush=True)
    app.run(host='0.0.0.0', port=8080)
