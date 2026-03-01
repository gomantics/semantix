package repositories_test

import (
	"net/http"
	"testing"

	approvals "github.com/approvals/go-approval-tests"
	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList_empty(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	body, err := s.Get("/v1/workspaces/" + wid + "/repos")
	require.NoError(t, err)
	assert.NotNil(t, body["repos"])
	assert.Equal(t, float64(0), body["total"])
}

func TestList_withItems(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	_, err := s.Post("/v1/workspaces/"+wid+"/repos", map[string]any{
		"url":    "https://github.com/example/listed-repo-" + testutil.UniqueID(),
		"branch": "main",
	})
	require.NoError(t, err)

	body, err := s.Get("/v1/workspaces/" + wid + "/repos")
	require.NoError(t, err)
	assert.Equal(t, float64(1), body["total"])
}

func TestList_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.GetStatus("/v1/workspaces/1/repos")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}

// TestList_approvals must NOT run in parallel (shared testdata file writes).
func TestList_approvals(t *testing.T) {
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	_, err := s.Post("/v1/workspaces/"+wid+"/repos", map[string]any{
		"url":    "https://github.com/example/approvals-list-repo",
		"branch": "main",
	})
	require.NoError(t, err)

	body, err := s.Get("/v1/workspaces/" + wid + "/repos")
	require.NoError(t, err)

	body["repos"] = "[SCRUBBED]"
	body["total"] = "[SCRUBBED]"
	approvals.VerifyJSONStruct(t, body)
}
