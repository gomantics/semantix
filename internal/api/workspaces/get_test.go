package workspaces_test

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

	uid := testutil.UniqueID()
	created, err := s.Post("/v1/workspaces", map[string]any{
		"name": "Get Workspace " + uid,
	})
	require.NoError(t, err)

	wid := fmt.Sprintf("%.0f", created["id"].(float64))
	body, err := s.Get("/v1/workspaces/" + wid)
	require.NoError(t, err)
	assert.Equal(t, created["id"], body["id"])
}

func TestGet_notFound(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.GetStatus("/v1/workspaces/999999999")
	testutil.RequireStatus(t, err, http.StatusNotFound)
}

func TestGet_invalidID(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.GetStatus("/v1/workspaces/not-a-number")
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}
