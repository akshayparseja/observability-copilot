# Observability Stack - Working Setup

## Components
- **OTel Collector**: Receives traces from apps, exports to Tempo (HTTP on 4318)
- **Tempo**: Stores traces (HTTP OTLP receiver on 4318, gRPC on 4317)
- **Grafana**: Visualizes traces from Tempo
- **Demo App**: Python Flask app with OTel instrumentation

## Key Fixes Applied
1. Tempo config: Added HTTP OTLP receiver on 4318
2. OTel Collector config: Uses otlphttp exporter to Tempo:4318
3. Demo app: Requirements.txt with correct OTel versions (0.59b0)
4. Dockerfile: Fixed to use k8s/demo-app/ path

## Files
- `k8s/tempo/tempo-configmap.yaml` - Tempo config with HTTP listener
- `k8s/otel-collector/otel-collector-config.yaml` - Collector config
- `k8s/demo-app/Dockerfile` - App Docker image
- `k8s/demo-app/demo-app.py` - OTel-instrumented Flask app
- `k8s/demo-app/requirements.txt` - Python dependencies

## Trace Flow
App → OTel Receiver (4317/4318) → OTel Collector → Tempo → Grafana UI

## Next Steps
- Add more instrumentation (databases, caching)
- Setup Prometheus metrics export
- Add Loki for logs
