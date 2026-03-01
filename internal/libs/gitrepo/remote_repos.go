package gitrepo

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gomantics/semantix/internal/libs/github"
	"github.com/gomantics/semantix/internal/libs/gitlab"
)

// RemoteRepo represents a repository accessible via a git token.
type RemoteRepo struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	URL           string `json:"url"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description,omitempty"`
}

// RemoteReposPage is a single page of RemoteRepo results.
// NextCursor is empty when there are no more pages.
type RemoteReposPage struct {
	Repos      []RemoteRepo
	NextCursor string
}

// HTTPClient is the HTTP client used by ListRemoteReposPage.
// Override in tests to point at a local httptest.Server.
var HTTPClient = &http.Client{
	Timeout: 15 * time.Second,
}

// ListRemoteReposPage returns one page of repositories accessible by the given
// token for the specified provider. Pass cursor="" for the first page; use
// RemoteReposPage.NextCursor for subsequent calls. The cursor encodes the
// provider page number for both GitHub and GitLab.
//
// When search is non-empty it is forwarded to the provider's API so that
// filtering happens server-side.
func ListRemoteReposPage(ctx context.Context, provider Provider, token, cursor string, perPage int, search string) (RemoteReposPage, error) {
	switch provider {
	case ProviderGitHub:
		return listGitHubReposPage(ctx, token, cursor, perPage, search)
	case ProviderGitLab:
		return listGitLabReposPage(ctx, token, cursor, perPage, search)
	default:
		return RemoteReposPage{}, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func listGitHubReposPage(ctx context.Context, token, cursor string, perPage int, search string) (RemoteReposPage, error) {
	page := 1
	if cursor != "" {
		var err error
		page, err = strconv.Atoi(cursor)
		if err != nil {
			return RemoteReposPage{}, fmt.Errorf("invalid cursor: %w", err)
		}
	}

	c := &github.Client{HTTPClient: HTTPClient}
	p, err := c.ListReposPage(ctx, token, page, perPage, search)
	if err != nil {
		return RemoteReposPage{}, fmt.Errorf("github: %w", err)
	}

	out := make([]RemoteRepo, len(p.Repos))
	for i, r := range p.Repos {
		out[i] = RemoteRepo{
			Name:          r.Name,
			FullName:      r.FullName,
			URL:           r.HTMLURL,
			Private:       r.Private,
			DefaultBranch: r.DefaultBranch,
			Description:   r.Description,
		}
	}

	nextCursor := ""
	if p.NextPage != 0 {
		nextCursor = strconv.Itoa(p.NextPage)
	}
	return RemoteReposPage{Repos: out, NextCursor: nextCursor}, nil
}

func listGitLabReposPage(ctx context.Context, token, cursor string, perPage int, search string) (RemoteReposPage, error) {
	page := 1
	if cursor != "" {
		var err error
		page, err = strconv.Atoi(cursor)
		if err != nil {
			return RemoteReposPage{}, fmt.Errorf("invalid cursor: %w", err)
		}
	}

	c := &gitlab.Client{HTTPClient: HTTPClient}
	p, err := c.ListProjectsPage(ctx, token, page, perPage, search)
	if err != nil {
		return RemoteReposPage{}, fmt.Errorf("gitlab: %w", err)
	}

	out := make([]RemoteRepo, len(p.Projects))
	for i, proj := range p.Projects {
		out[i] = RemoteRepo{
			Name:          proj.Name,
			FullName:      proj.PathWithNamespace,
			URL:           proj.WebURL,
			Private:       proj.Visibility != "public",
			DefaultBranch: proj.DefaultBranch,
			Description:   proj.Description,
		}
	}

	nextCursor := ""
	if p.NextPage != 0 {
		nextCursor = strconv.Itoa(p.NextPage)
	}
	return RemoteReposPage{Repos: out, NextCursor: nextCursor}, nil
}
