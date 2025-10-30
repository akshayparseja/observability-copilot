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
	}
	
	result.HasMetrics = detectMetrics(clonePath)
	result.HasOTel = detectOTel(clonePath)
	
	os.RemoveAll(clonePath)
	return result, nil
}

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

func detectMetrics(path string) bool {
	content, _ := os.ReadFile(filepath.Join(path, "requirements.txt"))
	return strings.Contains(string(content), "prometheus-client")
}

func detectOTel(path string) bool {
	content, _ := os.ReadFile(filepath.Join(path, "requirements.txt"))
	return strings.Contains(string(content), "opentelemetry")
}
