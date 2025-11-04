package togglespec

import "fmt"

func GenerateToggleSpec(serviceName, framework string, hasMetrics, hasOTel bool) (telemetryMode, spec string) {
    if hasMetrics && hasOTel {
        telemetryMode = "both"
        spec = fmt.Sprintf(`# ToggleSpec for %s
telemetry_mode: both
metrics:
  enabled: true
tracing:
  enabled: true
`, serviceName)
    } else if hasMetrics {
        telemetryMode = "metrics"
        spec = fmt.Sprintf(`# ToggleSpec for %s
telemetry_mode: metrics
metrics:
  enabled: true
tracing:
  enabled: false
`, serviceName)
    } else if hasOTel {
        telemetryMode = "traces"
        spec = fmt.Sprintf(`# ToggleSpec for %s
telemetry_mode: traces
metrics:
  enabled: false
tracing:
  enabled: true
`, serviceName)
    } else {
        telemetryMode = "none"
        spec = fmt.Sprintf(`# ToggleSpec for %s
telemetry_mode: none
metrics:
  enabled: false
tracing:
  enabled: false
`, serviceName)
    }
    return
}
