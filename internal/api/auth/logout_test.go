package auth_test

import (
	"net/http"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestLogout_clearsSession must not be parallel.
func TestLogout_clearsSession(t *testing.T) {
	getAdminState(t)
	s := testutil.NewState(t)
	_, err := s.Post("/v1/auth/login", map[string]any{
		"email":    adminEmail,
		"password": "password123",
	})
	require.NoError(t, err)

	// Verify authenticated.
	_, err = s.Get("/v1/auth/me")
	require.NoError(t, err)

	// Logout clears the cookie.
	err = s.PostStatus("/v1/auth/logout", nil)
	testutil.RequireStatus(t, err, http.StatusNoContent)

	// Now /me should return 401.
	err = s.GetStatus("/v1/auth/me")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}

func TestLogout_withoutSession(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	// Logout with no cookie should return 204 gracefully.
	err := s.PostStatus("/v1/auth/logout", nil)
	testutil.RequireStatus(t, err, http.StatusNoContent)
}
