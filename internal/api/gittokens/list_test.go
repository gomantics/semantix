package gittokens_test

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

	body, err := s.Get("/v1/gittokens")
	require.NoError(t, err)
	assert.NotNil(t, body["tokens"])
}

func TestList_withItems(t *testing.T) {
	t.Parallel()
	s := testutil.NewAuthState(t)

	uid := testutil.UniqueID()
	_, err := s.Post("/v1/gittokens", map[string]any{
		"name":     "Listed Token " + uid,
		"provider": "github",
		"token":    "ghp_test1234567890abcd",
	})
	require.NoError(t, err)

	body, err := s.Get("/v1/gittokens")
	require.NoError(t, err)
	tokens := body["tokens"].([]any)
	assert.NotEmpty(t, tokens)
}

func TestList_requiresAuth(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.GetStatus("/v1/gittokens")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}

// TestList_approvals must NOT run in parallel (shared testdata file writes).
func TestList_approvals(t *testing.T) {
	s := testutil.NewAuthState(t)

	_, err := s.Post("/v1/gittokens", map[string]any{
		"name":     "Approvals List Token",
		"provider": "gitlab",
		"token":    "glpat_test1234567890abcd",
	})
	require.NoError(t, err)

	body, err := s.Get("/v1/gittokens")
	require.NoError(t, err)

	// Scrub the tokens array since IDs/timestamps/hints are non-deterministic.
	body["tokens"] = "[SCRUBBED]"
	approvals.VerifyJSONStruct(t, body)
}
