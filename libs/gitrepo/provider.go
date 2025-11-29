package gitrepo

import (
	"github.com/go-git/go-git/v5/plumbing/transport"
)

// Provider defines the interface for git hosting services (GitHub, GitLab, Bitbucket, etc.)
type Provider interface {
	// Name returns the provider name (e.g., "github", "gitlab", "bitbucket")
	Name() string

	// NormalizeURL converts various URL formats to a standard clone URL
	NormalizeURL(url string) string

	// ParseURL extracts owner and repository name from a URL
	ParseURL(url string) (owner, repo string)

	// ValidateURL checks if the URL is valid for this provider
	ValidateURL(url string) error

	// Auth returns the authentication method for this provider (nil if no auth)
	Auth() transport.AuthMethod

	// MatchesURL returns true if the URL belongs to this provider
	MatchesURL(url string) bool
}

// Registry holds registered providers and allows auto-detection
type Registry struct {
	providers []Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make([]Provider, 0),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(p Provider) {
	r.providers = append(r.providers, p)
}

// Detect finds the appropriate provider for a given URL
func (r *Registry) Detect(url string) Provider {
	for _, p := range r.providers {
		if p.MatchesURL(url) {
			return p
		}
	}
	return nil
}

// Get returns a provider by name
func (r *Registry) Get(name string) Provider {
	for _, p := range r.providers {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

// DefaultRegistry is the global provider registry with common providers pre-registered
// Note: Providers are registered without authentication - use GetProviderWithToken for authenticated access
var DefaultRegistry = NewRegistry()

func init() {
	// Register default providers without authentication
	// Authentication is handled per-request using GetProviderWithToken
	DefaultRegistry.Register(NewGitHubProvider(""))
}

// GetProviderWithToken returns a provider with the given token for authentication
func GetProviderWithToken(providerName string, token string) Provider {
	switch providerName {
	case "github":
		return NewGitHubProvider(token)
	case "gitlab":
		// TODO: Add GitLab provider when needed
		return nil
	default:
		return nil
	}
}

// GetProviderForURL returns a provider for the given URL with optional token
func GetProviderForURL(url string, token string) Provider {
	baseProvider := DefaultRegistry.Detect(url)
	if baseProvider == nil {
		return nil
	}

	if token == "" {
		return baseProvider
	}

	return GetProviderWithToken(baseProvider.Name(), token)
}
