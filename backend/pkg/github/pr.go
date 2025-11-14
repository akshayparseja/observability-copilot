package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"observability-copilot/pkg/generator"
)

type PRRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

type PRResponse struct {
	HTMLURL string `json:"html_url"`
	Number  int    `json:"number"`
}

// CreateInstrumentationPR creates a PR with only missing instrumentation
func CreateInstrumentationPR(
	repoURL string,
	plan *generator.InstrumentationPlan,
	hasMetrics bool,
	hasOtel bool,
) (string, error) {

	// Get GitHub token first
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return "", fmt.Errorf("GITHUB_TOKEN not set")
	}

	// Parse repo owner and name from URL
	owner, repo := parseRepoURL(repoURL)
	if owner == "" || repo == "" {
		return "", fmt.Errorf("invalid repo URL: %s", repoURL)
	}

	// Clone repo
	tmpDir := filepath.Join("/tmp", fmt.Sprintf("%s-%s", owner, repo))
	os.RemoveAll(tmpDir)

	cmd := exec.Command("git", "clone", repoURL, tmpDir)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git clone failed: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Configure git user (REQUIRED to fix exit code 128)
	cmd = exec.Command("git", "-C", tmpDir, "config", "user.name", "Observability Copilot Bot")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git config user.name failed: %w", err)
	}

	cmd = exec.Command("git", "-C", tmpDir, "config", "user.email", "bot@observability-copilot.dev")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git config user.email failed: %w", err)
	}

	// Create branch name based on what we're adding
	branchName := getBranchName(plan.Mode, hasMetrics, hasOtel)

	// Create and checkout new branch
	cmd = exec.Command("git", "-C", tmpDir, "checkout", "-b", branchName)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git checkout failed: %w", err)
	}

	// Apply changes from plan
	for _, change := range plan.Changes {
		filePath := filepath.Join(tmpDir, change.Path)

		if change.Action == "append" {
			// Append to existing file
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return "", fmt.Errorf("failed to open %s: %w", change.Path, err)
			}
			_, err = f.WriteString(change.Content)
			f.Close()
			if err != nil {
				return "", err
			}
		} else if change.Action == "create" {
			// Create new file
			err := os.WriteFile(filePath, []byte(change.Content), 0644)
			if err != nil {
				return "", err
			}
		}
	}

	// Git add
	cmd = exec.Command("git", "-C", tmpDir, "add", ".")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git add failed: %w", err)
	}

	// Git commit
	commitMsg := getCommitMessage(plan.Mode, hasMetrics, hasOtel)
	cmd = exec.Command("git", "-C", tmpDir, "commit", "-m", commitMsg)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git commit failed: %w", err)
	}

	// Update remote URL with token for authentication
	authenticatedURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", token, owner, repo)
	cmd = exec.Command("git", "-C", tmpDir, "remote", "set-url", "origin", authenticatedURL)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to set remote URL: %w", err)
	}

	// Git push
	cmd = exec.Command("git", "-C", tmpDir, "push", "-u", "origin", branchName)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git push failed: %w", err)
	}

	// Create PR via GitHub API
	prURL, err := createGitHubPR(owner, repo, branchName, commitMsg, plan, token)
	if err != nil {
		return "", fmt.Errorf("failed to create PR: %w", err)
	}

	return prURL, nil
}

func parseRepoURL(url string) (owner, repo string) {
	// https://github.com/owner/repo.git -> owner, repo
	parts := strings.Split(strings.TrimSuffix(url, ".git"), "/")
	if len(parts) >= 2 {
		owner = parts[len(parts)-2]
		repo = parts[len(parts)-1]
	}
	return
}

func getBranchName(mode string, hasMetrics, hasOtel bool) string {
	if mode == "both" {
		if hasMetrics && !hasOtel {
			return "feat/add-opentelemetry-traces"
		} else if !hasMetrics && hasOtel {
			return "feat/add-prometheus-metrics"
		} else if !hasMetrics && !hasOtel {
			return "feat/add-observability"
		}
	}

	if mode == "metrics" && !hasMetrics {
		return "feat/add-prometheus-metrics"
	}

	if mode == "traces" && !hasOtel {
		return "feat/add-opentelemetry-traces"
	}

	return "feat/add-observability"
}

func getCommitMessage(mode string, hasMetrics, hasOtel bool) string {
	if mode == "both" {
		if hasMetrics && !hasOtel {
			return "feat: Add OpenTelemetry distributed tracing"
		} else if !hasMetrics && hasOtel {
			return "feat: Add Prometheus metrics instrumentation"
		}
		return "feat: Add observability with Prometheus and OpenTelemetry"
	}

	if mode == "metrics" {
		return "feat: Add Prometheus metrics instrumentation"
	}

	if mode == "traces" {
		return "feat: Add OpenTelemetry distributed tracing"
	}

	return "feat: Add observability instrumentation"
}

func createGitHubPR(owner, repo, branch, title string, plan *generator.InstrumentationPlan, token string) (string, error) {
	prReq := PRRequest{
		Title: title,
		Body:  generatePRBody(plan),
		Head:  branch,
		Base:  "main", // or "master" - you could make this configurable
	}

	body, _ := json.Marshal(prReq)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error: %s", string(bodyBytes))
	}

	var prResp PRResponse
	json.NewDecoder(resp.Body).Decode(&prResp)

	return prResp.HTMLURL, nil
}

func generatePRBody(plan *generator.InstrumentationPlan) string {
	body := fmt.Sprintf(`## 🔭 Observability Instrumentation

This PR adds **%s** instrumentation to your service.

### Changes Made:
`, plan.Mode)

	for _, change := range plan.Changes {
		body += fmt.Sprintf("- Modified `%s` to add %s\n", change.Path, change.Action)
	}

	body += `
### What's Included:
`
	if strings.Contains(plan.Mode, "metrics") || plan.Mode == "both" {
		body += "- ✅ Prometheus metrics endpoint (`/metrics`)\n"
		body += "- ✅ HTTP request counters and histograms\n"
	}

	if strings.Contains(plan.Mode, "traces") || plan.Mode == "both" {
		body += "- ✅ OpenTelemetry distributed tracing\n"
		body += "- ✅ Automatic span creation for HTTP requests\n"
		body += "- ✅ Integration with OTel Collector\n"
	}

	body += `
### Next Steps:
1. Review the changes
2. Test locally
3. Merge when ready

**Generated by Observability Copilot** 🚀
`
	return body
}
