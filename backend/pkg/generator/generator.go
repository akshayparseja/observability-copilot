package generator

import (
	"fmt"
	"observability-copilot/pkg/scanner"
)

type FileChange struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Action    string `json:"action"`
	LineAfter string `json:"line_after"`
}

type InstrumentationPlan struct {
	RepoID      string       `json:"repo_id"`
	Framework   string       `json:"framework"`
	Service     string       `json:"service"`
	Mode        string       `json:"mode"`
	Changes     []FileChange `json:"changes"`
	Description string       `json:"description"`
}

func Generate(framework, service, mode string, candidates []scanner.Candidate) (*InstrumentationPlan, error) {
	switch framework {
	case "Go":
		return generateGoInstrumentation(service, mode, candidates)
	case "Python":
		return generatePythonInstrumentation(service, mode)
	case "Java":
		return generateJavaInstrumentation(service, mode)
	case "Node.js":
		return generateNodeInstrumentation(service, mode)
	default:
		return nil, fmt.Errorf("unsupported framework: %s", framework)
	}
}

func generateNodeInstrumentation(service, mode string) (*InstrumentationPlan, error) {
	return nil, fmt.Errorf("Node.js instrumentation not implemented yet")
}
