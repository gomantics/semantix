package repos

// Status represents the indexing status of a repository
type Status string

const (
	StatusPending   Status = "pending"
	StatusCloning   Status = "cloning"
	StatusIndexing  Status = "indexing"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// String returns the string representation of the status
func (s Status) String() string {
	return string(s)
}

// IsActive returns true if the repo is currently being processed
func (s Status) IsActive() bool {
	return s == StatusCloning || s == StatusIndexing
}
