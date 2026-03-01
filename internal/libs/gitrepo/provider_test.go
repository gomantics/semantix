package gitrepo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected Provider
		wantErr  bool
	}{
		{"github", ProviderGitHub, false},
		{"GITHUB", ProviderGitHub, false},
		{"  github  ", ProviderGitHub, false},
		{"gitlab", ProviderGitLab, false},
		{"bitbucket", ProviderBitbucket, false},
		{"unknown", ProviderUnknown, true},
		{"invalid", ProviderUnknown, true},
		{"", ProviderUnknown, true},
	}

	for _, tt := range tests {
		name := tt.input
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseProvider(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDetectProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url      string
		expected Provider
	}{
		{"https://github.com/org/repo", ProviderGitHub},
		{"https://github.com/org/repo.git", ProviderGitHub},
		{"git@github.com:org/repo.git", ProviderGitHub},
		{"https://GITHUB.COM/org/repo", ProviderGitHub},
		{"https://gitlab.com/org/repo", ProviderGitLab},
		{"https://gitlab.com/org/subgroup/repo.git", ProviderGitLab},
		{"https://bitbucket.org/org/repo", ProviderBitbucket},
		{"https://bitbucket.org/org/repo.git", ProviderBitbucket},
		{"https://example.com/org/repo", ProviderUnknown},
		{"https://mygitlab.internal/org/repo", ProviderUnknown},
		{"", ProviderUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			got := DetectProvider(tt.url)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestAuthenticatedURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		repoURL  string
		token    string
		provider Provider
		wantUser string
		wantPass string
	}{
		{
			name:     "github injects x-access-token user",
			repoURL:  "https://github.com/org/repo",
			token:    "ghp_abc123",
			provider: ProviderGitHub,
			wantUser: "x-access-token",
			wantPass: "ghp_abc123",
		},
		{
			name:     "gitlab injects oauth2 user",
			repoURL:  "https://gitlab.com/org/repo",
			token:    "glpat-abc123",
			provider: ProviderGitLab,
			wantUser: "oauth2",
			wantPass: "glpat-abc123",
		},
		{
			name:     "bitbucket injects x-token-auth user",
			repoURL:  "https://bitbucket.org/org/repo",
			token:    "atl_abc123",
			provider: ProviderBitbucket,
			wantUser: "x-token-auth",
			wantPass: "atl_abc123",
		},
		{
			name:     "unknown provider uses generic token user",
			repoURL:  "https://example.com/org/repo",
			token:    "mytoken",
			provider: ProviderUnknown,
			wantUser: "token",
			wantPass: "mytoken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := AuthenticatedURL(tt.repoURL, tt.token, tt.provider)
			require.NoError(t, err)
			assert.Contains(t, got, tt.wantUser)
			assert.Contains(t, got, tt.wantPass)
		})
	}

	t.Run("empty token returns original URL unchanged", func(t *testing.T) {
		t.Parallel()
		url := "https://github.com/org/repo"
		got, err := AuthenticatedURL(url, "", ProviderGitHub)
		require.NoError(t, err)
		assert.Equal(t, url, got)
	})

	t.Run("invalid URL returns error", func(t *testing.T) {
		t.Parallel()
		_, err := AuthenticatedURL("://bad url", "token", ProviderGitHub)
		assert.Error(t, err)
	})
}
