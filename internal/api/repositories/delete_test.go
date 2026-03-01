package repositories_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestDelete_success(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	created, err := s.Post("/v1/workspaces/"+wid+"/repos", map[string]any{
		"url":    "https://github.com/example/del-repo-" + testutil.UniqueID(),
		"branch": "main",
	})
	require.NoError(t, err)

	rid := fmt.Sprintf("%.0f", created["id"].(float64))
	err = s.DeleteStatus("/v1/workspaces/" + wid + "/repos/" + rid)
	testutil.RequireStatus(t, err, http.StatusNoContent)
}

func TestDelete_notFound(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	err := s.DeleteStatus("/v1/workspaces/" + wid + "/repos/999999999")
	testutil.RequireStatus(t, err, http.StatusNotFound)
}

func TestDelete_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.DeleteStatus("/v1/workspaces/1/repos/1")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}
