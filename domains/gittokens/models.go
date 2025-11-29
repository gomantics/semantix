package gittokens

// Provider represents a supported git provider
type Provider string

const (
	ProviderGitHub Provider = "github"
	ProviderGitLab Provider = "gitlab"
)

// GitToken represents a git token for authenticating with git providers
type GitToken struct {
	ID          int64    `json:"id"`
	WorkspaceID int64    `json:"workspaceId"`
	Provider    Provider `json:"provider"`
	Name        string   `json:"name"`
	Created     int64    `json:"created"`
	Updated     int64    `json:"updated"`
}

// CreateParams are the parameters for creating a git token
type CreateParams struct {
	WorkspaceID int64
	Provider    Provider
	Name        string
	Token       string // Plain text token - will be encrypted before storage
}

// UpdateParams are the parameters for updating a git token
type UpdateParams struct {
	Name  string
	Token string // Optional - if empty, token is not updated
}

// ListParams are the parameters for listing git tokens
type ListParams struct {
	WorkspaceID int64
	Provider    *Provider // Optional filter by provider
}
