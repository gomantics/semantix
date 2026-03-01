package gittokens_test

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

func TestDelete_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.DeleteStatus("/v1/gittokens/1")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}
