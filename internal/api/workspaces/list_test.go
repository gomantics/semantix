package workspaces_test

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

	body, err := s.Get("/v1/workspaces")
	require.NoError(t, err)
	assert.NotNil(t, body["workspaces"])
}

func TestList_withItems(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	uid := testutil.UniqueID()
	_, err := s.Post("/v1/workspaces", map[string]any{
		"name": "Listed Workspace " + uid,
	})
	require.NoError(t, err)

	body, err := s.Get("/v1/workspaces")
	require.NoError(t, err)
	assert.NotNil(t, body["total"])
}

func TestList_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.GetStatus("/v1/workspaces")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}

// TestList_approvals must NOT run in parallel (shared testdata file writes).
func TestList_approvals(t *testing.T) {
	s := testutil.NewAuthState(t)

	uid := testutil.UniqueID()
	_, err := s.Post("/v1/workspaces", map[string]any{
		"name": "Approvals List Workspace " + uid,
	})
	require.NoError(t, err)

	body, err := s.Get("/v1/workspaces")
	require.NoError(t, err)

	body["workspaces"] = "[SCRUBBED]"
	body["total"] = "[SCRUBBED]"
	approvals.VerifyJSONStruct(t, body)
}
