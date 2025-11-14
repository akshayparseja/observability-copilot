package generator

import "fmt"

func generateJavaInstrumentation(service, mode string) (*InstrumentationPlan, error) {
    plan := &InstrumentationPlan{
        Framework:   "Java",
        Service:     service,
        Mode:        mode,
        Description: fmt.Sprintf("Add OpenTelemetry instrumentation for %s (mode: %s)", service, mode),
    }

    // Add dependencies to pom.xml
    if mode == "traces" || mode == "both" {
        plan.Changes = append(plan.Changes, FileChange{
            Path:   "pom.xml",
            Action: "append",
            Content: `
<!-- OpenTelemetry dependencies -->
<dependency>
    <groupId>io.opentelemetry</groupId>
    <artifactId>opentelemetry-api</artifactId>
    <version>1.32.0</version>
</dependency>
<dependency>
    <groupId>io.opentelemetry</groupId>
    <artifactId>opentelemetry-sdk</artifactId>
    <version>1.32.0</version>
</dependency>
<dependency>
    <groupId>io.opentelemetry</groupId>
    <artifactId>opentelemetry-exporter-otlp</artifactId>
    <version>1.32.0</version>
</dependency>
<dependency>
    <groupId>io.opentelemetry.instrumentation</groupId>
    <artifactId>opentelemetry-spring-boot-starter</artifactId>
    <version>2.0.0</version>
</dependency>`,
            LineAfter: "<dependencies>",
        })

        plan.Changes = append(plan.Changes, generateJavaTracerConfig(service))
    }

    if mode == "metrics" || mode == "both" {
        plan.Changes = append(plan.Changes, FileChange{
            Path:   "pom.xml",
            Action: "append",
            Content: `
<!-- Micrometer Prometheus dependencies -->
<dependency>
    <groupId>io.micrometer</groupId>
    <artifactId>micrometer-registry-prometheus</artifactId>
    <version>1.12.0</version>
</dependency>
<dependency>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-actuator</artifactId>
</dependency>`,
            LineAfter: "<dependencies>",
        })

        plan.Changes = append(plan.Changes, generateJavaMetricsConfig(service))
    }

    return plan, nil
}

func generateJavaTracerConfig(service string) FileChange {
    code := fmt.Sprintf(`# OpenTelemetry Configuration
# Add to src/main/resources/application.properties

# Service name
otel.service.name=%s

# OTLP exporter configuration
otel.traces.exporter=otlp
otel.exporter.otlp.endpoint=http://otel-collector.observability.svc.cluster.local:4317
otel.exporter.otlp.protocol=grpc

# Enable auto-instrumentation
otel.instrumentation.spring-boot.enabled=true
otel.instrumentation.spring-webmvc.enabled=true
otel.instrumentation.spring-web.enabled=true

# Sampling (always on for development, adjust for production)
otel.traces.sampler=always_on

# Log level
logging.level.io.opentelemetry=INFO
`, service)

    return FileChange{
        Path:    "src/main/resources/application-otel.properties",
        Action:  "create",
        Content: code,
    }
}

func generateJavaMetricsConfig(service string) FileChange {
    code := `# Prometheus Metrics Configuration
# Add to src/main/resources/application.properties

# Enable actuator endpoints
management.endpoints.web.exposure.include=health,prometheus,metrics
management.endpoint.prometheus.enabled=true
management.endpoint.metrics.enabled=true

# Prometheus endpoint
management.metrics.export.prometheus.enabled=true

# Metrics tags
management.metrics.tags.application=${spring.application.name}
management.metrics.tags.environment=${spring.profiles.active:dev}

# Enable common metrics
management.metrics.enable.jvm=true
management.metrics.enable.process=true
management.metrics.enable.system=true
management.metrics.enable.http.server.requests=true

# Actuator base path (metrics available at /actuator/prometheus)
management.endpoints.web.base-path=/actuator
`

    return FileChange{
        Path:    "src/main/resources/application-metrics.properties",
        Action:  "create",
        Content: code,
    }
}
