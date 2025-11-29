package gitrepo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/gomantics/semantix/config"
	"go.uber.org/zap"
)

// RepoMetadata contains information about a cloned repository
type RepoMetadata struct {
	Name          string
	Owner         string
	DefaultBranch string
	HeadCommitSHA string
	Provider      string
}

// FileInfo contains information about a file in the repository
type FileInfo struct {
	Path      string
	Shasum    string
	SizeBytes int64
	Language  string
}

// Clone clones a git repository to the specified destination
func Clone(ctx context.Context, l *zap.Logger, provider Provider, repoURL string, destPath string) (*git.Repository, error) {
	// Normalize URL using the provider
	url := provider.NormalizeURL(repoURL)

	l.Info("cloning repository",
		zap.String("provider", provider.Name()),
		zap.String("url", url),
		zap.String("dest", destPath),
	)

	// Prepare clone options
	opts := &git.CloneOptions{
		URL:      url,
		Progress: nil, // Could use os.Stdout for progress in dev mode
		Depth:    1,   // Shallow clone for efficiency
	}

	// Add authentication if provider has it configured
	if auth := provider.Auth(); auth != nil {
		opts.Auth = auth
	}

	// Clone the repository
	repo, err := git.PlainCloneContext(ctx, destPath, false, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	l.Info("repository cloned successfully")
	return repo, nil
}

// CloneWithAutoDetect clones a repository, auto-detecting the provider from the URL
func CloneWithAutoDetect(ctx context.Context, l *zap.Logger, repoURL string, destPath string) (*git.Repository, Provider, error) {
	provider := DefaultRegistry.Detect(repoURL)
	if provider == nil {
		return nil, nil, fmt.Errorf("unsupported git provider for URL: %s", repoURL)
	}

	repo, err := Clone(ctx, l, provider, repoURL, destPath)
	return repo, provider, err
}

// GetMetadata extracts metadata from a cloned repository
func GetMetadata(repo *git.Repository, provider Provider, repoURL string) (*RepoMetadata, error) {
	// Parse owner and name from URL using provider
	owner, name := provider.ParseURL(repoURL)

	// Get HEAD reference
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get default branch name
	branchName := "main"
	if head.Name().IsBranch() {
		branchName = head.Name().Short()
	}

	return &RepoMetadata{
		Name:          name,
		Owner:         owner,
		DefaultBranch: branchName,
		HeadCommitSHA: head.Hash().String(),
		Provider:      provider.Name(),
	}, nil
}

// ListFiles returns all files in the repository that should be indexed
func ListFiles(repoPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip hidden directories and common non-code directories
			name := info.Name()
			if strings.HasPrefix(name, ".") || isSkippedDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Skip files that are too large
		if info.Size() > config.Indexing.MaxFileSizeBytes() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return err
		}

		// Skip binary and non-text files
		if !isIndexableFile(relPath) {
			return nil
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

// GetFileInfo computes file information including shasum
func GetFileInfo(repoPath, filePath string) (*FileInfo, error) {
	fullPath := filepath.Join(repoPath, filePath)

	// Get file info
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	// Compute shasum
	shasum, err := ComputeShasum(fullPath)
	if err != nil {
		return nil, err
	}

	// Detect language
	language := detectLanguage(filePath)

	return &FileInfo{
		Path:      filePath,
		Shasum:    shasum,
		SizeBytes: info.Size(),
		Language:  language,
	}, nil
}

// ComputeShasum computes the SHA256 hash of a file
func ComputeShasum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ReadFile reads the content of a file
func ReadFile(repoPath, filePath string) (string, error) {
	fullPath := filepath.Join(repoPath, filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// CleanupRepo removes a cloned repository
func CleanupRepo(repoPath string) error {
	return os.RemoveAll(repoPath)
}

// GetRepoPath returns the path where a repository should be cloned
func GetRepoPath(repoID int64) string {
	return filepath.Join(config.Indexing.CloneDir(), fmt.Sprintf("%d", repoID))
}

// ValidateRepoURL validates that the URL is supported by a registered provider
func ValidateRepoURL(url string) error {
	provider := DefaultRegistry.Detect(url)
	if provider == nil {
		return fmt.Errorf("unsupported git provider for URL: %s", url)
	}
	return provider.ValidateURL(url)
}

// isSkippedDir returns true if the directory should be skipped
func isSkippedDir(name string) bool {
	skipDirs := map[string]bool{
		"node_modules":  true,
		"vendor":        true,
		"dist":          true,
		"build":         true,
		"target":        true,
		"__pycache__":   true,
		".git":          true,
		".svn":          true,
		".hg":           true,
		"coverage":      true,
		".idea":         true,
		".vscode":       true,
		"bin":           true,
		"obj":           true,
		".cache":        true,
		".pytest_cache": true,
		".mypy_cache":   true,
		"venv":          true,
		".venv":         true,
		"env":           true,
		".env":          true,
		"deps":          true,
		"_deps":         true,
		"third_party":   true,
		"external":      true,
		"packages":      true,
		".nuget":        true,
		".gradle":       true,
		".cargo":        true,
		"cmake-build":   true,
		"out":           true,
		"output":        true,
		".terraform":    true,
		".next":         true,
		".turbo":        true,
	}
	return skipDirs[name]
}

// isIndexableFile returns true if the file should be indexed
func isIndexableFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	// Common code file extensions
	codeExtensions := map[string]bool{
		".go":      true,
		".py":      true,
		".js":      true,
		".ts":      true,
		".jsx":     true,
		".tsx":     true,
		".rs":      true,
		".java":    true,
		".kt":      true,
		".scala":   true,
		".c":       true,
		".cpp":     true,
		".cc":      true,
		".cxx":     true,
		".h":       true,
		".hpp":     true,
		".cs":      true,
		".rb":      true,
		".php":     true,
		".swift":   true,
		".m":       true,
		".mm":      true,
		".lua":     true,
		".pl":      true,
		".pm":      true,
		".r":       true,
		".R":       true,
		".jl":      true,
		".ex":      true,
		".exs":     true,
		".erl":     true,
		".hrl":     true,
		".clj":     true,
		".cljs":    true,
		".hs":      true,
		".ml":      true,
		".mli":     true,
		".fs":      true,
		".fsx":     true,
		".dart":    true,
		".elm":     true,
		".vue":     true,
		".svelte":  true,
		".astro":   true,
		".sql":     true,
		".sh":      true,
		".bash":    true,
		".zsh":     true,
		".fish":    true,
		".ps1":     true,
		".bat":     true,
		".cmd":     true,
		".yaml":    true,
		".yml":     true,
		".toml":    true,
		".json":    true,
		".xml":     true,
		".html":    true,
		".htm":     true,
		".css":     true,
		".scss":    true,
		".sass":    true,
		".less":    true,
		".md":      true,
		".mdx":     true,
		".rst":     true,
		".txt":     true,
		".proto":   true,
		".graphql": true,
		".gql":     true,
		".tf":      true,
		".tfvars":  true,
		".nix":     true,
		".zig":     true,
		".nim":     true,
		".v":       true,
		".sol":     true,
		".move":    true,
	}

	return codeExtensions[ext]
}

// detectLanguage detects the programming language based on file extension
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	languageMap := map[string]string{
		".go":      "go",
		".py":      "python",
		".js":      "javascript",
		".ts":      "typescript",
		".jsx":     "javascript",
		".tsx":     "typescript",
		".rs":      "rust",
		".java":    "java",
		".kt":      "kotlin",
		".scala":   "scala",
		".c":       "c",
		".cpp":     "cpp",
		".cc":      "cpp",
		".cxx":     "cpp",
		".h":       "c",
		".hpp":     "cpp",
		".cs":      "csharp",
		".rb":      "ruby",
		".php":     "php",
		".swift":   "swift",
		".m":       "objective-c",
		".mm":      "objective-c",
		".lua":     "lua",
		".pl":      "perl",
		".pm":      "perl",
		".r":       "r",
		".R":       "r",
		".jl":      "julia",
		".ex":      "elixir",
		".exs":     "elixir",
		".erl":     "erlang",
		".hrl":     "erlang",
		".clj":     "clojure",
		".cljs":    "clojure",
		".hs":      "haskell",
		".ml":      "ocaml",
		".mli":     "ocaml",
		".fs":      "fsharp",
		".fsx":     "fsharp",
		".dart":    "dart",
		".elm":     "elm",
		".vue":     "vue",
		".svelte":  "svelte",
		".astro":   "astro",
		".sql":     "sql",
		".sh":      "shell",
		".bash":    "shell",
		".zsh":     "shell",
		".fish":    "shell",
		".ps1":     "powershell",
		".bat":     "batch",
		".cmd":     "batch",
		".yaml":    "yaml",
		".yml":     "yaml",
		".toml":    "toml",
		".json":    "json",
		".xml":     "xml",
		".html":    "html",
		".htm":     "html",
		".css":     "css",
		".scss":    "scss",
		".sass":    "sass",
		".less":    "less",
		".md":      "markdown",
		".mdx":     "markdown",
		".rst":     "rst",
		".txt":     "text",
		".proto":   "protobuf",
		".graphql": "graphql",
		".gql":     "graphql",
		".tf":      "terraform",
		".tfvars":  "terraform",
		".nix":     "nix",
		".zig":     "zig",
		".nim":     "nim",
		".v":       "vlang",
		".sol":     "solidity",
		".move":    "move",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}
	return "unknown"
}
