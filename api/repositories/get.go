package repositories

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/api/web"
	"github.com/gomantics/semantix/domains/repos"
	"go.uber.org/zap"
)

// GetResponse is the response for getting a repository
type GetResponse struct {
	ID            int64  `json:"id"`
	WorkspaceID   int64  `json:"workspace_id"`
	GitTokenID    *int64 `json:"git_token_id,omitempty"`
	URL           string `json:"url"`
	Name          string `json:"name"`
	Owner         string `json:"owner"`
	DefaultBranch string `json:"default_branch"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
	LastCommitSHA string `json:"last_commit_sha,omitempty"`
	FileCount     int64  `json:"file_count"`
	ChunkCount    int64  `json:"chunk_count"`
	Created       int64  `json:"created"`
	Updated       int64  `json:"updated"`
}

// Get handles GET /v1/repositories/:id
func Get(c web.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid repository id")
	}

	repo, err := repos.GetByIDWithStats(ctx, id)
	if errors.Is(err, repos.ErrNotFound) {
		return c.NotFound("repository not found")
	}
	if err != nil {
		c.L.Error("failed to get repo", zap.Error(err))
		return c.InternalError("failed to get repository")
	}

	return c.OK(GetResponse{
		ID:            repo.ID,
		WorkspaceID:   repo.WorkspaceID,
		GitTokenID:    repo.GitTokenID,
		URL:           repo.URL,
		Name:          repo.Name,
		Owner:         repo.Owner,
		DefaultBranch: repo.DefaultBranch,
		Status:        repo.Status.String(),
		Error:         repo.Error,
		LastCommitSHA: repo.LastCommitSHA,
		FileCount:     repo.FileCount,
		ChunkCount:    repo.ChunkCount,
		Created:       repo.Created,
		Updated:       repo.Updated,
	})
}
