package repositories_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_success(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	created, err := s.Post("/v1/workspaces/"+wid+"/repos", map[string]any{
		"url":    "https://github.com/example/get-repo-" + testutil.UniqueID(),
		"branch": "main",
	})
	require.NoError(t, err)

	rid := fmt.Sprintf("%.0f", created["id"].(float64))
	body, err := s.Get("/v1/workspaces/" + wid + "/repos/" + rid)
	require.NoError(t, err)
	assert.Equal(t, created["id"], body["id"])
}

func TestGet_notFound(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)
	wid := createWorkspace(t, s)

	err := s.GetStatus("/v1/workspaces/" + wid + "/repos/999999999")
	testutil.RequireStatus(t, err, http.StatusNotFound)
}

func TestGet_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.GetStatus("/v1/workspaces/1/repos/1")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}
