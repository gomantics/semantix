package auth_test

import (
	"net/http"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSignup_success(t *testing.T) {
	getAdminState(t)
	assert.Equal(t, adminEmail, adminSignupBody["email"])
	assert.NotEmpty(t, adminSignupBody["id"])
}

func TestSignup_missingEmail(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.PostStatus("/v1/auth/signup", map[string]any{
		"email":    "",
		"password": "password123",
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestSignup_shortPassword(t *testing.T) {
	t.Parallel()
	s := testutil.NewState(t)

	err := s.PostStatus("/v1/auth/signup", map[string]any{
		"email":    "short-pw-" + testutil.UniqueID() + "@test.com",
		"password": "short",
	})
	testutil.RequireStatus(t, err, http.StatusBadRequest)
}

func TestSignup_secondUserForbidden(t *testing.T) {
	getAdminState(t)
	s := testutil.NewState(t)

	err := s.PostStatus("/v1/auth/signup", map[string]any{
		"email":    "second-" + testutil.UniqueID() + "@test.com",
		"password": "password123",
	})
	testutil.RequireStatus(t, err, http.StatusForbidden)
}
