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
	baseBranch string,
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

	// Create and checkout new branch based on the selected baseBranch (if provided)
	if baseBranch == "" {
		baseBranch = "main"
	}
	// Ensure we have the base branch available locally
	cmd = exec.Command("git", "-C", tmpDir, "fetch", "origin", baseBranch)
	_ = cmd.Run()
	// Create branch from baseBranch
	cmd = exec.Command("git", "-C", tmpDir, "checkout", "-b", branchName, baseBranch)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git checkout failed: %w", err)
	}

	// Apply changes from plan
	for _, change := range plan.Changes {
		filePath := filepath.Join(tmpDir, change.Path)

		switch change.Action {
		case "append":
			// If LineAfter provided, insert after the matching line; else append to file
			if change.LineAfter != "" {
				if err := insertAfterLine(filePath, change.LineAfter, change.Content); err != nil {
					return "", fmt.Errorf("failed to modify %s: %w", change.Path, err)
				}
			} else {
				f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return "", fmt.Errorf("failed to open %s: %w", change.Path, err)
				}
				_, err = f.WriteString(change.Content)
				f.Close()
				if err != nil {
					return "", err
				}
			}
		case "create":
			// Create new file (and parent dirs)
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return "", fmt.Errorf("failed to create dirs for %s: %w", change.Path, err)
			}
			err := os.WriteFile(filePath, []byte(change.Content), 0644)
			if err != nil {
				return "", err
			}
		case "modify":
			// Modify: insert after a matching line or replace a marker
			if change.LineAfter == "" {
				return "", fmt.Errorf("modify action requires LineAfter for %s", change.Path)
			}
			if err := insertAfterLine(filePath, change.LineAfter, change.Content); err != nil {
				return "", fmt.Errorf("failed to modify %s: %w", change.Path, err)
			}
		default:
			return "", fmt.Errorf("unknown action: %s", change.Action)
		}
	}

	// Run gofmt and build to validate changes before committing
	cmd = exec.Command("gofmt", "-w", ".")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("gofmt failed: %v, output: %s", err, string(out))
	}

	cmd = exec.Command("go", "build", "./...")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("go build failed: %v, output: %s", err, string(out))
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
	prURL, err := createGitHubPR(owner, repo, branchName, commitMsg, plan, baseBranch, token)
	if err != nil {
		return "", fmt.Errorf("failed to create PR: %w", err)
	}

	return prURL, nil
}

// insertAfterLine inserts `content` after the first occurrence of `matchLine` in filePath.
// If matchLine is not found, it appends the content at the end.
func insertAfterLine(filePath, matchLine, content string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	var outLines []string
	inserted := false
	for _, line := range lines {
		outLines = append(outLines, line)
		if !inserted && strings.Contains(line, matchLine) {
			// Insert content after this line
			outLines = append(outLines, content)
			inserted = true
		}
	}
	if !inserted {
		outLines = append(outLines, content)
	}
	newData := strings.Join(outLines, "\n")
	return os.WriteFile(filePath, []byte(newData), 0644)
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

func createGitHubPR(owner, repo, branch, title string, plan *generator.InstrumentationPlan, baseBranch string, token string) (string, error) {
	if baseBranch == "" {
		baseBranch = "main"
	}
	prReq := PRRequest{
		Title: title,
		Body:  generatePRBody(plan),
		Head:  branch,
		Base:  baseBranch,
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
