package github

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
)

// ListBranches returns remote branch names for the given repo URL.
// If token is non-empty it will be used to construct an authenticated URL
// so private repos can be listed. If token is empty, GITHUB_TOKEN env var
// will be used if present.
func ListBranches(repoURL, token string) ([]string, error) {
    if token == "" {
        token = os.Getenv("GITHUB_TOKEN")
    }

    remote := repoURL
    // If token provided and repoURL is an https GitHub URL, inject token for auth
    if token != "" && strings.HasPrefix(repoURL, "https://") && strings.Contains(repoURL, "github.com") {
        // e.g. https://github.com/owner/repo.git -> https://x-access-token:TOKEN@github.com/owner/repo.git
        remote = strings.Replace(repoURL, "https://", fmt.Sprintf("https://x-access-token:%s@", token), 1)
    }

    cmd := exec.Command("git", "ls-remote", "--heads", remote)
    out, err := cmd.Output()
    if err != nil {
        // try without token in case injection broke URL
        if remote != repoURL {
            cmd2 := exec.Command("git", "ls-remote", "--heads", repoURL)
            out2, err2 := cmd2.Output()
            if err2 == nil {
                out = out2
                err = nil
            }
        }
    }
    if err != nil {
        return nil, fmt.Errorf("git ls-remote failed: %w", err)
    }

    lines := strings.Split(strings.TrimSpace(string(out)), "\n")
    var branches []string
    for _, line := range lines {
        if line == "" {
            continue
        }
        parts := strings.Split(line, "\t")
        if len(parts) != 2 {
            continue
        }
        ref := parts[1]
        // refs/heads/<branch>
        if strings.HasPrefix(ref, "refs/heads/") {
            branches = append(branches, strings.TrimPrefix(ref, "refs/heads/"))
        }
    }
    return branches, nil
}
