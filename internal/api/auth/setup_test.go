package auth_test

import (
	"os"
	"sync"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	main := testutil.Main(m,
		testutil.WithPostgres(),
	)
	os.Exit(main.Run())
}

// adminState holds the single authenticated session available in this package.
// Because the signup endpoint only allows one user per DB, we create it once
// across all tests using sync.Once.
var (
	adminOnce       sync.Once
	adminEmail      string
	adminState      *testutil.State
	adminSignupBody map[string]any
)

// getAdminState returns the shared authenticated State, creating the admin
// user via signup on the first call. Safe to call from any test regardless
// of file ordering.
func getAdminState(t *testing.T) *testutil.State {
	t.Helper()
	adminOnce.Do(func() {
		s := testutil.NewState(t)
		email := "admin-" + testutil.UniqueID() + "@test.com"
		body, err := s.Post("/v1/auth/signup", map[string]any{
			"email":    email,
			"password": "password123",
		})
		require.NoError(t, err)
		adminEmail = email
		adminState = s
		adminSignupBody = body
	})
	return adminState
}
