package search_test

import (
	"os"
	"testing"

	"github.com/gomantics/semantix/internal/testutil"
)

func TestMain(m *testing.M) {
	main := testutil.Main(m,
		testutil.WithPostgres(),
		testutil.WithAdminUser(),
	)
	os.Exit(main.Run())
}
