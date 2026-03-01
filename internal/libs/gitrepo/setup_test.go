package gitrepo

import (
	"os"
	"testing"

	_ "github.com/gomantics/semantix/internal/testflags"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
