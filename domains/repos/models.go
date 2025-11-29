package repos

// Repo represents a repository in the domain layer
type Repo struct {
	ID            int64
	WorkspaceID   int64
	GitTokenID    *int64
	URL           string
	Name          string
	Owner         string
	DefaultBranch string
	LastCommitSHA string
	Status        Status
	Error         string
	Created       int64
	Updated       int64
}

// CreateParams contains parameters for creating a repository
type CreateParams struct {
	WorkspaceID int64
	GitTokenID  *int64
	URL         string
}

// ListParams contains parameters for listing repositories
type ListParams struct {
	WorkspaceID int64
	Status      *Status
	Limit       int
	Offset      int
}

// ListResult contains the result of listing repositories
type ListResult struct {
	Repos []Repo
	Total int64
}

// RepoWithStats includes file and chunk counts
type RepoWithStats struct {
	Repo
	FileCount  int64
	ChunkCount int64
}
