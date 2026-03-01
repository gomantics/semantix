package workspaces_test

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

	uid := testutil.UniqueID()
	created, err := s.Post("/v1/workspaces", map[string]any{
		"name": "Delete Workspace " + uid,
	})
	require.NoError(t, err)

	wid := fmt.Sprintf("%.0f", created["id"].(float64))
	err = s.DeleteStatus("/v1/workspaces/" + wid)
	testutil.RequireStatus(t, err, http.StatusNoContent)
}

func TestDelete_notFound(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.DeleteStatus("/v1/workspaces/999999999")
	testutil.RequireStatus(t, err, http.StatusNotFound)
}

func TestDelete_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.DeleteStatus("/v1/workspaces/1")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}
