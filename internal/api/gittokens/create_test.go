package gittokens_test

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
	body, err := s.Post("/v1/gittokens", map[string]any{
		"name":     "My Token " + uid,
		"provider": "github",
		"token":    "ghp_test1234567890abcd",
	})
	require.NoError(t, err)
	assert.Equal(t, "github", body["provider"])
	assert.Equal(t, "My Token "+uid, body["name"])
	assert.Nil(t, body["token"], "raw token must not be exposed")
}

func TestCreate_missingName(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.PostStatus("/v1/gittokens", map[string]any{
		"name":     "",
		"provider": "github",
		"token":    "ghp_test1234567890abcd",
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestCreate_missingProvider(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.PostStatus("/v1/gittokens", map[string]any{
		"name":     "Token " + testutil.UniqueID(),
		"provider": "",
		"token":    "ghp_test1234567890abcd",
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestCreate_missingToken(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	err := s.PostStatus("/v1/gittokens", map[string]any{
		"name":     "Token " + testutil.UniqueID(),
		"provider": "github",
		"token":    "",
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestCreate_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.PostStatus("/v1/gittokens", map[string]any{
		"name":     "Token",
		"provider": "github",
		"token":    "ghp_test1234567890abcd",
	})
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}

// TestCreate_approvals must NOT run in parallel (shared testdata file writes).
func TestCreate_approvals(t *testing.T) {
	s := testutil.NewAuthState(t)

	body, err := s.Post("/v1/gittokens", map[string]any{
		"name":     "Approvals Token",
		"provider": "github",
		"token":    "ghp_test1234567890abcd",
	})
	require.NoError(t, err)

	testutil.ScrubFields(body, "id", "created", "hint")
	approvals.VerifyJSONStruct(t, body)
}
