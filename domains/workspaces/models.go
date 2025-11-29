package workspaces

// Workspace represents a workspace for organizing repositories
type Workspace struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
}

// CreateParams are the parameters for creating a workspace
type CreateParams struct {
	Name string
	Slug string
}

// UpdateParams are the parameters for updating a workspace
type UpdateParams struct {
	Name string
	Slug string
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
