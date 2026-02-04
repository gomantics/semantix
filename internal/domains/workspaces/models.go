package workspaces

// Workspace represents a workspace for organizing repositories
type Workspace struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	Slug        string            `json:"slug"`
	Description *string           `json:"description,omitempty"`
	Settings    map[string]any    `json:"settings"`
	Created     int64             `json:"created"`
	Updated     int64             `json:"updated"`
}

// CreateParams are the parameters for creating a workspace
type CreateParams struct {
	Name        string
	Slug        string
	Description *string
	Settings    map[string]any
}

// UpdateParams are the parameters for updating a workspace
type UpdateParams struct {
	Name        string
	Slug        string
	Description *string
	Settings    map[string]any
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
