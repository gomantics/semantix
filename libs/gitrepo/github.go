package gitrepo

import (
	"fmt"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// GitHubProvider implements Provider for GitHub repositories
type GitHubProvider struct {
	pat string // Personal Access Token
}

// NewGitHubProvider creates a new GitHub provider with optional PAT authentication
func NewGitHubProvider(pat string) *GitHubProvider {
	return &GitHubProvider{pat: pat}
}

func (g *GitHubProvider) Name() string {
	return "github"
}

func (g *GitHubProvider) NormalizeURL(url string) string {
	// Remove trailing .git if present
	url = strings.TrimSuffix(url, ".git")

	// Handle various formats
	switch {
	case strings.HasPrefix(url, "git@github.com:"):
		// SSH format: git@github.com:owner/repo
		url = strings.Replace(url, "git@github.com:", "https://github.com/", 1)
	case strings.HasPrefix(url, "github.com/"):
		// Short format: github.com/owner/repo
		url = "https://" + url
	case !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://"):
		// Assume it's just owner/repo
		url = "https://github.com/" + url
	}

	return url + ".git"
}

func (g *GitHubProvider) ParseURL(url string) (owner, repo string) {
	// Normalize and strip common prefixes
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimPrefix(url, "https://github.com/")
	url = strings.TrimPrefix(url, "http://github.com/")
	url = strings.TrimPrefix(url, "github.com/")
	url = strings.TrimPrefix(url, "git@github.com:")

	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return "", url
}

func (g *GitHubProvider) ValidateURL(url string) error {
	owner, name := g.ParseURL(url)
	if owner == "" || name == "" {
		return fmt.Errorf("invalid GitHub repository URL format: %s", url)
	}
	return nil
}

func (g *GitHubProvider) Auth() transport.AuthMethod {
	if g.pat == "" {
		return nil
	}
	return &http.BasicAuth{
		Username: "git", // GitHub uses "git" as username for token auth
		Password: g.pat,
	}
}

func (g *GitHubProvider) MatchesURL(url string) bool {
	url = strings.ToLower(url)
	return strings.Contains(url, "github.com") ||
		strings.HasPrefix(url, "git@github.com:")
}

// SetPAT updates the Personal Access Token
func (g *GitHubProvider) SetPAT(pat string) {
	g.pat = pat
}
