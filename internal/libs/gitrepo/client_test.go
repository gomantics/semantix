package gitrepo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldExclude(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     string
		excluded bool
	}{
		// Excluded directories
		{"node_modules", true},
		{"node_modules/lodash/index.js", true},
		{"src/node_modules/package/file.js", true},
		{"vendor", true},
		{"vendor/github.com/pkg/foo.go", true},
		{".git", true},
		{".git/config", true},
		{"dist", true},
		{"dist/bundle.js", true},
		{"build", true},
		{"build/output.js", true},

		// Excluded by *.lock pattern (suffix must be ".lock")
		{"yarn.lock", true},
		{"Gemfile.lock", true},
		{"Pipfile.lock", true},
		// Excluded by *.min.js pattern
		{"app.min.js", true},
		{"vendor/assets/app.min.js", true},
		// Not excluded - package-lock.json ends in .json not .lock
		{"package-lock.json", false},
		// Not excluded - go.sum is not a lock file pattern
		{"go.sum", false},

		// Not excluded
		{"main.go", false},
		{"src/main.go", false},
		{"internal/db/init.go", false},
		{"README.md", false},
		{"Makefile", false},
		{"app.js", false},
		{"builder.go", false},
		{"distribution.go", false},
		{"cmd/api/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got := ShouldExclude(tt.path)
			assert.Equal(t, tt.excluded, got, "ShouldExclude(%q)", tt.path)
		})
	}
}

func TestRepoDir(t *testing.T) {
	t.Parallel()

	got := RepoDir("/tmp/repos", 42, 7)
	assert.Equal(t, "/tmp/repos/42/7", got)

	got = RepoDir("./clones", 1, 100)
	assert.Equal(t, "clones/1/100", got)
}

func TestFetchSpecs(t *testing.T) {
	t.Parallel()

	specs := FetchSpecs("main")
	assert.Len(t, specs, 1)
	assert.Contains(t, string(specs[0]), "main")
}
