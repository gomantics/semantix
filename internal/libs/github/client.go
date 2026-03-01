// Package github provides a client for the GitHub REST API.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

const (
	defaultTimeout   = 15 * time.Second
	maxResponseBytes = 5 << 20 // 5 MiB
	apiBaseURL       = "https://api.github.com"
	apiVersionHeader = "2022-11-28"
	acceptHeader     = "application/vnd.github+json"
	fetchPageSize    = 100
	maxFetchPages    = 10
)

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
// When search is non-empty all accessible repos are fetched from /user/repos
// and filtered client-side so that org repos are included in results.
func (c *Client) ListReposPage(ctx context.Context, token string, page, perPage int, search string) (Page, error) {
	if search != "" {
		return c.searchReposPage(ctx, token, perPage, search)
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

// searchReposPage fetches repos from /user/repos in batches and filters
// client-side by name/description. This avoids the search API which returns
// all public repos on GitHub.
func (c *Client) searchReposPage(ctx context.Context, token string, perPage int, search string) (Page, error) {
	lower := strings.ToLower(search)
	var matched []Repo

	for p := 1; p <= maxFetchPages; p++ {
		u := fmt.Sprintf("%s/user/repos?per_page=%d&page=%d", apiBaseURL, fetchPageSize, p)

		var rows []repo
		if err := c.doJSON(ctx, token, u, &rows); err != nil {
			return Page{}, err
		}

		for _, r := range rows {
			nameLower := strings.ToLower(r.Name)
			descLower := strings.ToLower(r.Description)
			if strings.Contains(nameLower, lower) || strings.Contains(descLower, lower) {
				matched = append(matched, convertRepo(r))
			}
		}

		if len(rows) < fetchPageSize {
			break
		}
	}

	sortByNameRelevance(matched, lower)

	if len(matched) > perPage {
		matched = matched[:perPage]
	}
	return Page{Repos: matched}, nil
}

func sortByNameRelevance(repos []Repo, lower string) {
	slices.SortStableFunc(repos, func(a, b Repo) int {
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

func convertRepo(r repo) Repo {
	return Repo{
		Name:          r.Name,
		FullName:      r.FullName,
		HTMLURL:       r.HTMLURL,
		Private:       r.Private,
		DefaultBranch: r.DefaultBranch,
		Description:   r.Description,
	}
}

func convertRepos(rows []repo) []Repo {
	out := make([]Repo, len(rows))
	for i, r := range rows {
		out[i] = convertRepo(r)
	}
	return out
}
