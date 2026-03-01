package repos

// Status represents a repository's current state in the indexing pipeline.
type Status string

const (
	StatusPending  Status = "pending"
	StatusCloning  Status = "cloning"
	StatusIndexing Status = "indexing"
	StatusReady    Status = "ready"
	StatusError    Status = "error"
)

// Repo represents a repository within a workspace.
type Repo struct {
	ID           int64   `json:"id"`
	WorkspaceID  int64   `json:"workspace_id"`
	URL          string  `json:"url"`
	Branch       string  `json:"branch"`
	Status       Status  `json:"status"`
	IndexedAt    *int64  `json:"indexed_at,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
	Created      int64   `json:"created"`
	Updated      int64   `json:"updated"`
}

type CreateParams struct {
	WorkspaceID int64
	URL         string
	Branch      string
}

type UpdateParams struct {
	URL    string
	Branch string
}

type ListParams struct {
	WorkspaceID int64
	Limit       int
	Offset      int
}

type ListResult struct {
	Repos []Repo
	Total int64
}
