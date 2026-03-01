// Package gitlab provides a client for the GitLab REST API.
package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const (
	defaultTimeout   = 15 * time.Second
	maxResponseBytes = 5 << 20 // 5 MiB
	apiBaseURL       = "https://gitlab.com/api/v4"
)

// project is the JSON shape returned by the projects endpoint.
type project struct {
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
	Visibility        string `json:"visibility"`
	DefaultBranch     string `json:"default_branch"`
	Description       string `json:"description"`
}

// Project is the GitLab-native representation of a project (repository).
type Project struct {
	Name              string
	PathWithNamespace string
	WebURL            string
	Visibility        string
	DefaultBranch     string
	Description       string
}

// Page is a single page of projects returned by ListProjectsPage.
// NextPage is 0 when there are no more pages.
type Page struct {
	Projects []Project
	NextPage int
}

// Client calls the GitLab REST API using the provided HTTPClient.
type Client struct {
	HTTPClient *http.Client
}

// New returns a Client with sensible production defaults.
func New() *Client {
	return &Client{HTTPClient: &http.Client{Timeout: defaultTimeout}}
}

// ListProjectsPage returns one page of projects the token has membership in.
// Pass page=1 to start; use Page.NextPage for subsequent calls (0 means done).
// When search is non-empty it is forwarded as the ?search= parameter so that
// filtering happens server-side.
func (c *Client) ListProjectsPage(ctx context.Context, token string, page, perPage int, search string) (Page, error) {
	u := fmt.Sprintf("%s/projects?membership=true&per_page=%d&page=%d", apiBaseURL, perPage, page)
	if search != "" {
		u += "&search=" + url.QueryEscape(search)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Page{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return Page{}, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Page{}, fmt.Errorf("gitlab API status %d: %s", resp.StatusCode, body)
	}

	var rows []project
	limited := io.LimitReader(resp.Body, maxResponseBytes)
	if err := json.NewDecoder(limited).Decode(&rows); err != nil {
		return Page{}, fmt.Errorf("decode response: %w", err)
	}

	out := make([]Project, len(rows))
	for i, p := range rows {
		out[i] = Project{
			Name:              p.Name,
			PathWithNamespace: p.PathWithNamespace,
			WebURL:            p.WebURL,
			Visibility:        p.Visibility,
			DefaultBranch:     p.DefaultBranch,
			Description:       p.Description,
		}
	}

	if search != "" {
		sortProjectsByNameRelevance(out, search)
	}

	nextPage := 0
	if len(rows) == perPage {
		nextPage = page + 1
	}
	return Page{Projects: out, NextPage: nextPage}, nil
}

func sortProjectsByNameRelevance(projects []Project, search string) {
	lower := strings.ToLower(search)
	slices.SortStableFunc(projects, func(a, b Project) int {
		aName := strings.ToLower(a.Name)
		bName := strings.ToLower(b.Name)
		aExact := aName == lower
		bExact := bName == lower
		if aExact != bExact {
			if aExact {
				return -1
			}
			return 1
		}
		aHas := strings.Contains(aName, lower)
		bHas := strings.Contains(bName, lower)
		if aHas != bHas {
			if aHas {
				return -1
			}
			return 1
		}
		return 0
	})
}
