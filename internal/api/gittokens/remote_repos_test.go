package gittokens_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	approvals "github.com/approvals/go-approval-tests"
	"github.com/gomantics/semantix/internal/libs/gitrepo"
	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/require"
)

// allRepos is the full set of repos the mock GitHub server knows about.
var allRepos = []map[string]any{
	{
		"name":           "my-repo",
		"full_name":      "octocat/my-repo",
		"html_url":       "https://github.com/octocat/my-repo",
		"private":        false,
		"default_branch": "main",
		"description":    "A test repo",
	},
	{
		"name":           "private-repo",
		"full_name":      "octocat/private-repo",
		"html_url":       "https://github.com/octocat/private-repo",
		"private":        true,
		"default_branch": "main",
		"description":    "",
	},
	{
		"name":           "third-repo",
		"full_name":      "octocat/third-repo",
		"html_url":       "https://github.com/octocat/third-repo",
		"private":        false,
		"default_branch": "main",
		"description":    "",
	},
}

// mockGitHub starts a test server that responds like the GitHub API.
// /user/repos:              page 1 returns two repos, page 2 returns one, beyond that empty.
// /search/repositories:    filters allRepos by the q= prefix (before " user:@me").
func mockGitHub(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/search/repositories" {
			q := r.URL.Query().Get("q")
			term := q
			if i := strings.Index(q, " user:"); i >= 0 {
				term = q[:i]
			}
			var matched []map[string]any
			for _, repo := range allRepos {
				name := repo["name"].(string)
				fullName := repo["full_name"].(string)
				if containsFold(name, term) || containsFold(fullName, term) {
					matched = append(matched, repo)
				}
			}
			if matched == nil {
				matched = []map[string]any{}
			}
			json.NewEncoder(w).Encode(map[string]any{
				"total_count": len(matched),
				"items":       matched,
			})
			return
		}

		// /user/repos - paginated list
		switch r.URL.Query().Get("page") {
		case "", "1":
			json.NewEncoder(w).Encode(allRepos[:2])
		case "2":
			json.NewEncoder(w).Encode(allRepos[2:3])
		default:
			json.NewEncoder(w).Encode([]any{})
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func containsFold(s, sub string) bool {
	if sub == "" {
		return true
	}
	sl, subl := len(s), len(sub)
	for i := 0; i <= sl-subl; i++ {
		if strings.EqualFold(s[i:i+subl], sub) {
			return true
		}
	}
	return false
}

// withMockHTTPClient temporarily replaces gitrepo.HTTPClient with one
// that redirects requests to the provided server, then restores the original.
func withMockHTTPClient(t *testing.T, srv *httptest.Server) {
	t.Helper()
	original := gitrepo.HTTPClient
	gitrepo.HTTPClient = &http.Client{
		Transport: redirectTransport(srv.URL),
	}
	t.Cleanup(func() { gitrepo.HTTPClient = original })
}

// redirectTransport rewrites the host of every request to the given base URL.
type redirectTransport string

func (base redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = string(base)[len("http://"):]
	return http.DefaultTransport.RoundTrip(cloned)
}

func createTokenForRemoteRepos(t *testing.T, s *testutil.State) string {
	t.Helper()
	uid := testutil.UniqueID()
	created, err := s.Post("/v1/gittokens", map[string]any{
		"name":     "Remote Repos Token " + uid,
		"provider": "github",
		"token":    "ghp_test1234567890abcd",
	})
	require.NoError(t, err)
	return fmt.Sprintf("%.0f", created["id"].(float64))
}

func TestListRemoteRepos_notFound(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.GetStatus("/v1/gittokens/999999999/remote-repos")
	testutil.RequireStatus(t, err, http.StatusNotFound)
}

func TestListRemoteRepos_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.GetStatus("/v1/gittokens/1/remote-repos")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}

func TestListRemoteRepos_invalidID(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.GetStatus("/v1/gittokens/not-a-number/remote-repos")
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestListRemoteRepos_invalidPerPage(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.GetStatus("/v1/gittokens/1/remote-repos?per_page=bad")
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

// TestListRemoteRepos_success must NOT run in parallel (shared gitrepo.HTTPClient global).
func TestListRemoteRepos_success(t *testing.T) {
	srv := mockGitHub(t)
	withMockHTTPClient(t, srv)

	s := testutil.NewAuthState(t)
	id := createTokenForRemoteRepos(t, s)

	body, err := s.Get("/v1/gittokens/" + id + "/remote-repos")
	require.NoError(t, err)

	repos := body["repos"].([]any)
	require.Len(t, repos, 2)
	require.Equal(t, float64(30), body["per_page"])
	// mock page 1 has exactly 2 repos which is less than per_page=30, so no next cursor
	_, hasNext := body["next_cursor"]
	require.False(t, hasNext)
}

// TestListRemoteRepos_pagination must NOT run in parallel (shared gitrepo.HTTPClient global).
func TestListRemoteRepos_pagination(t *testing.T) {
	srv := mockGitHub(t)
	withMockHTTPClient(t, srv)

	s := testutil.NewAuthState(t)
	id := createTokenForRemoteRepos(t, s)

	// page 1 with per_page=2: gets 2 repos, exactly fills page, so next_cursor is set
	body, err := s.Get("/v1/gittokens/" + id + "/remote-repos?per_page=2")
	require.NoError(t, err)
	require.Equal(t, float64(2), body["per_page"])
	repos := body["repos"].([]any)
	require.Len(t, repos, 2)
	nextCursor, ok := body["next_cursor"].(string)
	require.True(t, ok, "expected next_cursor to be present")
	require.NotEmpty(t, nextCursor)

	// follow the cursor: gets the third repo, which is less than per_page=2, so no next cursor
	body, err = s.Get("/v1/gittokens/" + id + "/remote-repos?per_page=2&cursor=" + nextCursor)
	require.NoError(t, err)
	repos = body["repos"].([]any)
	require.Len(t, repos, 1)
	_, hasNext := body["next_cursor"]
	require.False(t, hasNext)
}

// TestListRemoteRepos_search must NOT run in parallel (shared gitrepo.HTTPClient global).
func TestListRemoteRepos_search(t *testing.T) {
	srv := mockGitHub(t)
	withMockHTTPClient(t, srv)

	s := testutil.NewAuthState(t)
	id := createTokenForRemoteRepos(t, s)

	// "private" matches only octocat/private-repo
	body, err := s.Get("/v1/gittokens/" + id + "/remote-repos?search=private")
	require.NoError(t, err)
	repos := body["repos"].([]any)
	require.Len(t, repos, 1)
	require.Equal(t, "private-repo", repos[0].(map[string]any)["name"])

	// "my" matches only octocat/my-repo
	body, err = s.Get("/v1/gittokens/" + id + "/remote-repos?search=my")
	require.NoError(t, err)
	repos = body["repos"].([]any)
	require.Len(t, repos, 1)
	require.Equal(t, "my-repo", repos[0].(map[string]any)["name"])

	// "nomatch" returns empty list
	body, err = s.Get("/v1/gittokens/" + id + "/remote-repos?search=nomatch")
	require.NoError(t, err)
	repos = body["repos"].([]any)
	require.Empty(t, repos)
}

// TestListRemoteRepos_invalidCursor must NOT run in parallel (shared gitrepo.HTTPClient global).
func TestListRemoteRepos_invalidCursor(t *testing.T) {
	srv := mockGitHub(t)
	withMockHTTPClient(t, srv)

	s := testutil.NewAuthState(t)
	id := createTokenForRemoteRepos(t, s)

	err := s.GetStatus("/v1/gittokens/" + id + "/remote-repos?cursor=notanumber")
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

// TestListRemoteRepos_upstreamError must NOT run in parallel (shared gitrepo.HTTPClient global).
func TestListRemoteRepos_upstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"internal error"}`))
	}))
	t.Cleanup(srv.Close)
	withMockHTTPClient(t, srv)

	s := testutil.NewAuthState(t)
	id := createTokenForRemoteRepos(t, s)

	err := s.GetStatus("/v1/gittokens/" + id + "/remote-repos")
	testutil.RequireStatus(t, err, http.StatusBadGateway)
}

// TestListRemoteRepos_upstreamTimeout must NOT run in parallel (shared gitrepo.HTTPClient global).
func TestListRemoteRepos_upstreamTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	original := gitrepo.HTTPClient
	gitrepo.HTTPClient = &http.Client{
		Timeout:   50 * time.Millisecond,
		Transport: redirectTransport(srv.URL),
	}
	t.Cleanup(func() { gitrepo.HTTPClient = original })

	s := testutil.NewAuthState(t)
	id := createTokenForRemoteRepos(t, s)

	err := s.GetStatus("/v1/gittokens/" + id + "/remote-repos")
	testutil.RequireStatus(t, err, http.StatusGatewayTimeout)
}

// TestListRemoteRepos_approvals must NOT run in parallel (shared testdata file writes).
func TestListRemoteRepos_approvals(t *testing.T) {
	srv := mockGitHub(t)
	withMockHTTPClient(t, srv)

	s := testutil.NewAuthState(t)
	id := createTokenForRemoteRepos(t, s)

	body, err := s.Get("/v1/gittokens/" + id + "/remote-repos")
	require.NoError(t, err)

	approvals.VerifyJSONStruct(t, body)
}
