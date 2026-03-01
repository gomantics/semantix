package gitrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Exclude patterns that should be sparse-checked out.
var sparseExcludes = []string{
	"node_modules/",
	"vendor/",
	".git/",
	"*.lock",
	"*.min.js",
	"dist/",
	"build/",
}

// CloneOptions configures a clone operation.
type CloneOptions struct {
	URL      string
	Branch   string
	Token    string
	Provider Provider
	DestDir  string
}

// Clone performs a shallow clone of the repository into destDir.
// If the repo already exists at destDir, it pulls the latest changes instead.
func Clone(ctx context.Context, opts CloneOptions) error {
	if _, err := os.Stat(filepath.Join(opts.DestDir, ".git")); err == nil {
		return pull(ctx, opts)
	}
	return cloneFresh(ctx, opts)
}

func cloneFresh(ctx context.Context, opts CloneOptions) error {
	cloneURL := opts.URL
	var auth *http.BasicAuth

	if opts.Token != "" {
		auth = &http.BasicAuth{
			Username: tokenUser(opts.Provider),
			Password: opts.Token,
		}
	}

	if !strings.HasSuffix(cloneURL, ".git") {
		cloneURL += ".git"
	}

	cloneOpts := &git.CloneOptions{
		URL:           cloneURL,
		Auth:          auth,
		Depth:         1,
		SingleBranch:  true,
		ReferenceName: plumbing.NewBranchReferenceName(opts.Branch),
		Tags:          git.NoTags,
	}

	_, err := git.PlainCloneContext(ctx, opts.DestDir, false, cloneOpts)
	if err != nil {
		return fmt.Errorf("clone %s: %w", opts.URL, err)
	}

	return nil
}

func pull(ctx context.Context, opts CloneOptions) error {
	repo, err := git.PlainOpen(opts.DestDir)
	if err != nil {
		return fmt.Errorf("open repo at %s: %w", opts.DestDir, err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	var auth *http.BasicAuth
	if opts.Token != "" {
		auth = &http.BasicAuth{
			Username: tokenUser(opts.Provider),
			Password: opts.Token,
		}
	}

	pullOpts := &git.PullOptions{
		RemoteName:    "origin",
		ReferenceName: plumbing.NewBranchReferenceName(opts.Branch),
		Auth:          auth,
		Depth:         1,
		SingleBranch:  true,
		Force:         true,
	}

	err = wt.PullContext(ctx, pullOpts)
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	if err != nil {
		return fmt.Errorf("pull %s: %w", opts.URL, err)
	}

	return nil
}

func tokenUser(provider Provider) string {
	switch provider {
	case ProviderGitHub:
		return "x-access-token"
	case ProviderGitLab:
		return "oauth2"
	default:
		return "token"
	}
}

// CheckConnectivity performs a lightweight ls-remote to verify that the given
// URL is reachable and the token (if provided) has read access. It does not
// clone or write anything to disk.
func CheckConnectivity(ctx context.Context, opts CloneOptions) error {
	cloneURL := opts.URL
	if !strings.HasSuffix(cloneURL, ".git") {
		cloneURL += ".git"
	}

	var auth *http.BasicAuth
	if opts.Token != "" {
		auth = &http.BasicAuth{
			Username: tokenUser(opts.Provider),
			Password: opts.Token,
		}
	}

	remote := git.NewRemote(nil, &config.RemoteConfig{
		Name: "origin",
		URLs: []string{cloneURL},
	})

	_, err := remote.ListContext(ctx, &git.ListOptions{Auth: auth})
	if err != nil {
		return fmt.Errorf("connectivity check failed for %s: %w", opts.URL, err)
	}
	return nil
}

// ShouldExclude returns true if the given relative path should be excluded
// from indexing (matches sparse checkout exclude patterns).
func ShouldExclude(relPath string) bool {
	for _, pattern := range sparseExcludes {
		if strings.HasSuffix(pattern, "/") {
			dir := strings.TrimSuffix(pattern, "/")
			if relPath == dir || strings.HasPrefix(relPath, dir+"/") || strings.Contains(relPath, "/"+dir+"/") {
				return true
			}
		} else if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(relPath, suffix) {
				return true
			}
		} else if relPath == pattern {
			return true
		}
	}
	return false
}

// RepoDir constructs the clone destination path for a repo.
func RepoDir(baseDir string, workspaceID, repoID int64) string {
	return filepath.Join(baseDir, fmt.Sprintf("%d", workspaceID), fmt.Sprintf("%d", repoID))
}

// FetchSpecs returns the default fetch refspec for sparse checkout configuration.
// This is exported for testing and advanced usage.
func FetchSpecs(branch string) []config.RefSpec {
	return []config.RefSpec{
		config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", branch, branch)),
	}
}
