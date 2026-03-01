// Package github provides a client for the GitHub REST API.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultTimeout   = 15 * time.Second
	maxResponseBytes = 5 << 20 // 5 MiB
	apiBaseURL       = "https://api.github.com"
	apiVersionHeader = "2022-11-28"
	acceptHeader     = "application/vnd.github+json"
)

// repo is the JSON shape returned by both the list and search endpoints.
type repo struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	HTMLURL       string `json:"html_url"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description"`
}

// Repo is the GitHub-native representation of a repository.
type Repo struct {
	Name          string
	FullName      string
	HTMLURL       string
	Private       bool
	DefaultBranch string
	Description   string
}

// Page is a single page of repositories returned by ListReposPage.
// NextPage is 0 when there are no more pages.
type Page struct {
	Repos    []Repo
	NextPage int
}

// Client calls the GitHub REST API using the provided HTTPClient.
type Client struct {
	HTTPClient *http.Client
}

// New returns a Client with sensible production defaults.
func New() *Client {
	return &Client{HTTPClient: &http.Client{Timeout: defaultTimeout}}
}

// ListReposPage returns one page of repositories accessible by the token.
// Pass page=1 to start; use Page.NextPage for subsequent calls (0 means done).
//
// When search is non-empty the GitHub Search API is used so that filtering
// happens server-side. When search is empty the standard list endpoint is used.
func (c *Client) ListReposPage(ctx context.Context, token string, page, perPage int, search string) (Page, error) {
	if search != "" {
		return c.searchReposPage(ctx, token, page, perPage, search)
	}
	return c.listReposPage(ctx, token, page, perPage)
}

func (c *Client) listReposPage(ctx context.Context, token string, page, perPage int) (Page, error) {
	u := fmt.Sprintf("%s/user/repos?per_page=%d&page=%d", apiBaseURL, perPage, page)

	var rows []repo
	if err := c.doJSON(ctx, token, u, &rows); err != nil {
		return Page{}, err
	}

	out := convertRepos(rows)
	nextPage := 0
	if len(rows) == perPage {
		nextPage = page + 1
	}
	return Page{Repos: out, NextPage: nextPage}, nil
}

func (c *Client) searchReposPage(ctx context.Context, token string, page, perPage int, search string) (Page, error) {
	type searchResponse struct {
		TotalCount int    `json:"total_count"`
		Items      []repo `json:"items"`
	}

	q := url.QueryEscape(search + " user:@me")
	u := fmt.Sprintf("%s/search/repositories?q=%s&per_page=%d&page=%d", apiBaseURL, q, perPage, page)

	var result searchResponse
	if err := c.doJSON(ctx, token, u, &result); err != nil {
		return Page{}, err
	}

	out := convertRepos(result.Items)
	nextPage := 0
	if len(result.Items) == perPage {
		nextPage = page + 1
	}
	return Page{Repos: out, NextPage: nextPage}, nil
}

// doJSON performs a GET request with standard GitHub headers and decodes the
// JSON response into dst. It enforces a response body size limit to prevent
// unbounded reads from a misbehaving upstream.
func (c *Client) doJSON(ctx context.Context, token, rawURL string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("X-GitHub-Api-Version", apiVersionHeader)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("github API status %d: %s", resp.StatusCode, body)
	}

	limited := io.LimitReader(resp.Body, maxResponseBytes)
	if err := json.NewDecoder(limited).Decode(dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func convertRepos(rows []repo) []Repo {
	out := make([]Repo, len(rows))
	for i, r := range rows {
		out[i] = Repo{
			Name:          r.Name,
			FullName:      r.FullName,
			HTMLURL:       r.HTMLURL,
			Private:       r.Private,
			DefaultBranch: r.DefaultBranch,
			Description:   r.Description,
		}
	}
	return out
}
