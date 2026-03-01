package auth_test

import (
	"net/http"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin_invalidCreds(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.PostStatus("/v1/auth/login", map[string]any{
		"email":    "nobody-" + testutil.UniqueID() + "@test.com",
		"password": "wrongpass",
	})
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}

func TestLogin_missingFields(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.PostStatus("/v1/auth/login", map[string]any{
		"email":    "",
		"password": "",
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

// TestLogin_validCredentials must not be parallel: depends on admin user existing.
func TestLogin_validCredentials(t *testing.T) {
	getAdminState(t)
	s := testutil.NewState(t)

	body, err := s.Post("/v1/auth/login", map[string]any{
		"email":    adminEmail,
		"password": "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, adminEmail, body["email"])
}
