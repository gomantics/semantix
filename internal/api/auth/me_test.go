package auth_test

import (
	"net/http"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMe_authenticated(t *testing.T) {
	getAdminState(t)
	s := testutil.NewState(t)

	_, err := s.Post("/v1/auth/login", map[string]any{
		"email":    adminEmail,
		"password": "password123",
	})
	require.NoError(t, err)

	body, err := s.Get("/v1/auth/me")
	require.NoError(t, err)
	assert.NotEmpty(t, body["id"])
	assert.Equal(t, adminEmail, body["email"])
}

func TestMe_unauthenticated(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.GetStatus("/v1/auth/me")
	testutil.RequireStatus(t, err, http.StatusUnauthorized)
}
