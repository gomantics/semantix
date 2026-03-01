package gittokens_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/require"
)

func createToken(t *testing.T, s *testutil.State) string {
	t.Helper()
	uid := testutil.UniqueID()
	body, err := s.Post("/v1/gittokens", map[string]any{
		"name":     "Token " + uid,
		"provider": "github",
		"token":    "ghp_test1234567890abcd",
	})
	require.NoError(t, err)
	return fmt.Sprintf("%.0f", body["id"].(float64))
}

func TestDelete_success(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	uid := testutil.UniqueID()
	created, err := s.Post("/v1/gittokens", map[string]any{
		"name":     "Delete Token " + uid,
		"provider": "github",
		"token":    "ghp_test1234567890abcd",
	})
	require.NoError(t, err)

	id := fmt.Sprintf("%.0f", created["id"].(float64))
	err = s.DeleteStatus("/v1/gittokens/" + id)
	testutil.RequireStatus(t, err, http.StatusNoContent)
}

func TestDelete_notFound(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.DeleteStatus("/v1/gittokens/999999999")
	testutil.RequireStatus(t, err, http.StatusNotFound)
}

func TestDelete_blockedWhenInUse(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	// Create a token
	uid := testutil.UniqueID()
	tokenBody, err := s.Post("/v1/gittokens", map[string]any{
		"name":     "In-Use Token " + uid,
		"provider": "github",
		"token":    "ghp_test1234567890abcd",
	})
	require.NoError(t, err)
	tokenIDFloat := tokenBody["id"].(float64)
	tokenID := fmt.Sprintf("%.0f", tokenIDFloat)

	// Create a workspace and a private repo referencing the token
	wsBody, err := s.Post("/v1/workspaces", map[string]any{
		"name": "Token In Use WS " + uid,
	})
	require.NoError(t, err)
	wid := fmt.Sprintf("%.0f", wsBody["id"].(float64))

	_, err = s.Post("/v1/workspaces/"+wid+"/repos", map[string]any{
		"url":          "https://github.com/example/private-" + uid,
		"is_private":   true,
		"git_token_id": tokenIDFloat,
	})
	require.NoError(t, err)

	// Delete should be blocked
	deleteErr := s.DeleteStatus("/v1/gittokens/" + tokenID)
	testutil.RequireStatus(t, deleteErr, http.StatusConflict)
}

func TestDelete_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.DeleteStatus("/v1/gittokens/1")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}
