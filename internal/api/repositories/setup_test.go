package repositories_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	main := testutil.Main(m,
		testutil.WithPostgres(),
		testutil.WithAdminUser(),
		testutil.WithApprovals(),
	)
	os.Exit(main.Run())
}

// createWorkspace is a helper that creates a workspace and returns its ID string.
func createWorkspace(t *testing.T, s *testutil.State) string {
	t.Helper()
	uid := testutil.UniqueID()
	body, err := s.Post("/v1/workspaces", map[string]any{
		"name": "Repo Test Workspace " + uid,
		"slug": "repo-test-ws-" + uid,
	})
	require.NoError(t, err)
	return fmt.Sprintf("%.0f", body["id"].(float64))
}
