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
	GitTokenID   *int64  `json:"git_token_id,omitempty"`
	URL          string  `json:"url"`
	Branch       string  `json:"branch"`
	IsPrivate    bool    `json:"is_private"`
	Status       Status  `json:"status"`
	IndexedAt    *int64  `json:"indexed_at,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
	Created      int64   `json:"created"`
	Updated      int64   `json:"updated"`
}

type CreateParams struct {
	WorkspaceID int64
	GitTokenID  *int64
	URL         string
	Branch      string
	IsPrivate   bool
}

type UpdateParams struct {
	GitTokenID *int64
	URL        string
	Branch     string
	IsPrivate  bool
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
