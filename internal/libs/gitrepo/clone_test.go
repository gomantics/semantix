package gitrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeLocalRepo creates a hermetic bare git repo in a temp directory with two
// committed files (main.go and README.md). Returns the file:// URL to the bare
// repo and the working directory path (for adding further commits in tests).
func makeLocalRepo(t *testing.T) (bareURL string, workDir string) {
	t.Helper()

	tmp := t.TempDir()
	bareDir := filepath.Join(tmp, "bare.git")
	workDir = filepath.Join(tmp, "work")

	// Create the bare repository.
	run(t, tmp, "git", "init", "--bare", bareDir)

	// Clone bare into a working directory.
	run(t, tmp, "git", "clone", bareDir, workDir)

	// Configure identity so commits work in any environment.
	run(t, workDir, "git", "config", "user.email", "test@test.com")
	run(t, workDir, "git", "config", "user.name", "Test")

	// Add initial files.
	writeFile(t, workDir, "main.go", "package main\n\nfunc main() {}\n")
	writeFile(t, workDir, "README.md", "# Test Repo\n")

	run(t, workDir, "git", "add", ".")
	run(t, workDir, "git", "commit", "-m", "initial commit")
	run(t, workDir, "git", "push", "origin", "HEAD:main")

	return "file://" + bareDir, workDir
}

// addCommit adds a new file and commits+pushes it to the bare repo.
func addCommit(t *testing.T, workDir, filename, content string) {
	t.Helper()

	writeFile(t, workDir, filename, content)
	run(t, workDir, "git", "add", ".")
	run(t, workDir, "git", "commit", "-m", "add "+filename)
	run(t, workDir, "git", "push", "origin", "HEAD:main")
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "command %s %v failed: %s", name, args, out)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
}

func TestClone_fresh(t *testing.T) {
	t.Parallel()
	bareURL, _ := makeLocalRepo(t)
	destDir := t.TempDir()

	err := Clone(context.Background(), CloneOptions{
		URL:      bareURL,
		Branch:   "main",
		DestDir:  destDir,
		Provider: ProviderUnknown,
	})
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(destDir, "main.go"))
	assert.FileExists(t, filepath.Join(destDir, "README.md"))
}

func TestClone_pull_alreadyUpToDate(t *testing.T) {
	t.Parallel()
	bareURL, _ := makeLocalRepo(t)
	destDir := t.TempDir()

	opts := CloneOptions{
		URL:      bareURL,
		Branch:   "main",
		DestDir:  destDir,
		Provider: ProviderUnknown,
	}

	// First clone.
	require.NoError(t, Clone(context.Background(), opts))

	// Second call should detect .git and pull; already up to date is swallowed.
	err := Clone(context.Background(), opts)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(destDir, "main.go"))
}

func TestClone_pull_withNewCommit(t *testing.T) {
	t.Parallel()
	bareURL, workDir := makeLocalRepo(t)
	destDir := t.TempDir()

	opts := CloneOptions{
		URL:      bareURL,
		Branch:   "main",
		DestDir:  destDir,
		Provider: ProviderUnknown,
	}

	// Initial clone.
	require.NoError(t, Clone(context.Background(), opts))
	assert.NoFileExists(t, filepath.Join(destDir, "new_file.go"))

	// Add a new commit to the upstream bare repo.
	addCommit(t, workDir, "new_file.go", "package main\n")

	// Pull should bring in the new file.
	err := Clone(context.Background(), opts)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(destDir, "new_file.go"))
}

func TestClone_badURL(t *testing.T) {
	t.Parallel()
	destDir := t.TempDir()

	err := Clone(context.Background(), CloneOptions{
		URL:      "file:///nonexistent/repo.git",
		Branch:   "main",
		DestDir:  destDir,
		Provider: ProviderUnknown,
	})

	assert.Error(t, err)
}
