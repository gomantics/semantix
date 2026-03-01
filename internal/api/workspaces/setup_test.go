package workspaces_test

import (
	"os"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
)

func TestMain(m *testing.M) {
	main := testutil.Main(m,
		testutil.WithPostgres(),
		testutil.WithAdminUser(),
		testutil.WithApprovals(),
	)
	os.Exit(main.Run())
}
