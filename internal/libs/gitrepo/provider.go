package gitrepo

import (
	"fmt"
	"net/url"
	"strings"
)

type Provider string

const (
	ProviderGitHub    Provider = "github"
	ProviderGitLab    Provider = "gitlab"
	ProviderBitbucket Provider = "bitbucket"
	ProviderUnknown   Provider = "unknown"
)

// DetectProvider determines the git hosting provider from a clone URL.
func DetectProvider(repoURL string) Provider {
	lower := strings.ToLower(repoURL)
	switch {
	case strings.Contains(lower, "github.com"):
		return ProviderGitHub
	case strings.Contains(lower, "gitlab.com"):
		return ProviderGitLab
	case strings.Contains(lower, "bitbucket.org"):
		return ProviderBitbucket
	default:
		return ProviderUnknown
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
	case ProviderBitbucket:
		parsed.User = url.UserPassword("x-token-auth", token)
	default:
		parsed.User = url.UserPassword("token", token)
	}

	return parsed.String(), nil
}
