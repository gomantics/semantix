package gitrepo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Cloner is the interface for cloning a git repository.
type Cloner interface {
	Clone(ctx context.Context, opts CloneOptions) error
}

// DefaultCloner uses go-git to perform real clones. It is the production implementation.
type DefaultCloner struct{}

func (d *DefaultCloner) Clone(ctx context.Context, opts CloneOptions) error {
	return Clone(ctx, opts)
}

// FakeCloner writes in-memory files to the destination directory instead of
// cloning a real repository. Intended for use in tests only.
type FakeCloner struct {
	// Files maps relative file paths to their content.
	Files map[string][]byte
	// Err, if non-nil, is returned from Clone instead of writing files.
	Err error
}

func (f *FakeCloner) Clone(_ context.Context, opts CloneOptions) error {
	if f.Err != nil {
		return fmt.Errorf("clone %s: %w", opts.URL, f.Err)
	}

	for relPath, content := range f.Files {
		dest := filepath.Join(opts.DestDir, relPath)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(dest), err)
		}
		if err := os.WriteFile(dest, content, 0644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
	}

	return nil
}
