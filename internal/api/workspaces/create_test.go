package workspaces_test

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

	uid := testutil.UniqueID()
	body, err := s.Post("/v1/workspaces", map[string]any{
		"name": "Test Workspace " + uid,
	})
	require.NoError(t, err)
	assert.Equal(t, "Test Workspace "+uid, body["name"])
}

func TestCreate_missingName(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.PostStatus("/v1/workspaces", map[string]any{
		"name": "",
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestCreate_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.PostStatus("/v1/workspaces", map[string]any{
		"name": "Test",
	})
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}

// TestCreate_approvals must NOT run in parallel (shared testdata file writes).
func TestCreate_approvals(t *testing.T) {
	s := testutil.NewAuthState(t)

	uid := testutil.UniqueID()
	body, err := s.Post("/v1/workspaces", map[string]any{
		"name": "Approvals Workspace " + uid,
	})
	require.NoError(t, err)

	testutil.ScrubFields(body, "id", "created", "updated")
	body["name"] = "[SCRUBBED]"
	approvals.VerifyJSONStruct(t, body)
}
