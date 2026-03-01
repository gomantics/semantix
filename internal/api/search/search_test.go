package search_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestSearch_notImplemented(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	// Create a workspace to use as the route parameter.
	uid := testutil.UniqueID()
	ws, err := s.Post("/v1/workspaces", map[string]any{
		"name": "Search Test Workspace " + uid,
	})
	require.NoError(t, err)
	wid := fmt.Sprintf("%.0f", ws["id"].(float64))

	err = s.PostStatus("/v1/workspaces/"+wid+"/search", map[string]any{
		"query": "find something",
		"limit": 10,
	})
	testutil.RequireStatus(t, err, http.StatusNotImplemented)
}

func TestSearch_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.PostStatus("/v1/workspaces/1/search", map[string]any{
		"query": "find something",
	})
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}
