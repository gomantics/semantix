package gitrepo

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

type Provider string

const (
	ProviderGitHub Provider = "github"
	ProviderGitLab Provider = "gitlab"
)

// ValidProviders are the providers that can be used for git tokens.
var ValidProviders = []Provider{ProviderGitHub, ProviderGitLab}

// ParseProvider validates and returns a Provider from a string.
// Only github and gitlab are accepted for token configuration.
func ParseProvider(s string) (Provider, error) {
	p := Provider(strings.ToLower(strings.TrimSpace(s)))
	if slices.Contains(ValidProviders, p) {
		return p, nil
	}
	return "", fmt.Errorf("invalid provider %q: must be github or gitlab", s)
}

// DetectProvider determines the git hosting provider from a clone URL.
// Returns an error if the URL does not match a supported provider.
func DetectProvider(repoURL string) (Provider, error) {
	lower := strings.ToLower(repoURL)
	switch {
	case strings.Contains(lower, "github.com"):
		return ProviderGitHub, nil
	case strings.Contains(lower, "gitlab.com"):
		return ProviderGitLab, nil
	default:
		return "", fmt.Errorf("unsupported git host in URL %q: must be github.com or gitlab.com", repoURL)
	}
}

// AuthenticatedURL injects a token into the clone URL for the given provider.
func AuthenticatedURL(repoURL, token string, provider Provider) (string, error) {
	if token == "" {
		return repoURL, nil
	}

	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid repository URL: %w", err)
	}

	switch provider {
	case ProviderGitHub:
		parsed.User = url.UserPassword("x-access-token", token)
	case ProviderGitLab:
		parsed.User = url.UserPassword("oauth2", token)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	return parsed.String(), nil
}
