package workspaces

// WorkspaceSettings configures indexing behavior for a workspace.
type WorkspaceSettings struct {
	// ExcludePatterns are glob patterns to exclude from indexing (e.g. ["docs/", "*.test.ts"]).
	ExcludePatterns []string `json:"exclude_patterns,omitempty"`
	// IncludePatterns restrict indexing to matching files (e.g. ["*.go", "*.ts", "*.py"]).
	IncludePatterns []string `json:"include_patterns,omitempty"`
}

// Workspace represents a workspace for organizing repositories
type Workspace struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description *string           `json:"description,omitempty"`
	Settings    WorkspaceSettings `json:"settings"`
	Created     int64             `json:"created"`
	Updated     int64             `json:"updated"`
}

// CreateParams are the parameters for creating a workspace
type CreateParams struct {
	Name        string
	Slug        string
	Description *string
	Settings    *WorkspaceSettings
}

// UpdateParams are the parameters for updating a workspace
type UpdateParams struct {
	Name        string
	Slug        string
	Description *string
	Settings    *WorkspaceSettings
}

// ListParams are the parameters for listing workspaces
type ListParams struct {
	Limit  int
	Offset int
}

// ListResult contains the result of listing workspaces
type ListResult struct {
	Workspaces []Workspace
	Total      int64
}
