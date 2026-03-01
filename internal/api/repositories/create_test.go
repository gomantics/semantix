package repositories_test

import (
	"net/http"
	"testing"

	approvals "github.com/approvals/go-approval-tests"
	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate_success(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	body, err := s.Post("/v1/workspaces/"+wid+"/repos", map[string]any{
		"url":    "https://github.com/example/repo-" + testutil.UniqueID(),
		"branch": "main",
	})
	require.NoError(t, err)
	assert.Equal(t, "pending", body["status"])
	assert.Equal(t, "main", body["branch"])
}

func TestCreate_privateWithoutToken(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	err := s.PostStatus("/v1/workspaces/"+wid+"/repos", map[string]any{
		"url":        "https://github.com/example/private-repo",
		"is_private": true,
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestCreate_missingURL(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	err := s.PostStatus("/v1/workspaces/"+wid+"/repos", map[string]any{
		"url": "",
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestCreate_invalidWorkspaceID(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.PostStatus("/v1/workspaces/not-a-number/repos", map[string]any{
		"url": "https://github.com/example/repo",
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestCreate_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.PostStatus("/v1/workspaces/1/repos", map[string]any{
		"url": "https://github.com/example/repo",
	})
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}

// TestCreate_approvals must NOT run in parallel (shared testdata file writes).
func TestCreate_approvals(t *testing.T) {
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	body, err := s.Post("/v1/workspaces/"+wid+"/repos", map[string]any{
		"url":    "https://github.com/example/approvals-repo",
		"branch": "main",
	})
	require.NoError(t, err)

	testutil.ScrubFields(body, "id", "workspace_id", "created", "updated")
	approvals.VerifyJSONStruct(t, body)
}
