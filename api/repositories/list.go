package repositories

import (
	"strconv"

	"github.com/gomantics/semantix/api/web"
	"github.com/gomantics/semantix/domains/repos"
	"go.uber.org/zap"
)

// ListResponse is the response for listing repositories
type ListResponse struct {
	Repos []RepoSummary `json:"repos"`
	Total int64         `json:"total"`
}

// RepoSummary is a summary of a repository
type RepoSummary struct {
	ID            int64  `json:"id"`
	WorkspaceID   int64  `json:"workspace_id"`
	URL           string `json:"url"`
	Name          string `json:"name"`
	Owner         string `json:"owner"`
	Status        string `json:"status"`
	LastCommitSHA string `json:"last_commit_sha,omitempty"`
	Created       int64  `json:"created"`
	Updated       int64  `json:"updated"`
}

// List handles GET /v1/repositories
func List(c web.Context) error {
	ctx := c.Request().Context()

	workspaceID, err := strconv.ParseInt(c.QueryParam("workspace_id"), 10, 64)
	if err != nil || workspaceID <= 0 {
		return c.BadRequest("workspace_id query parameter is required")
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	params := repos.ListParams{
		WorkspaceID: workspaceID,
		Limit:       limit,
		Offset:      (page - 1) * limit,
	}

	if status := c.QueryParam("status"); status != "" {
		s := repos.Status(status)
		params.Status = &s
	}

	result, err := repos.List(ctx, params)
	if err != nil {
		c.L.Error("failed to list repos", zap.Error(err))
		return c.InternalError("failed to list repositories")
	}

	summaries := make([]RepoSummary, len(result.Repos))
	for i, repo := range result.Repos {
		summaries[i] = RepoSummary{
			ID:            repo.ID,
			WorkspaceID:   repo.WorkspaceID,
			URL:           repo.URL,
			Name:          repo.Name,
			Owner:         repo.Owner,
			Status:        repo.Status.String(),
			LastCommitSHA: repo.LastCommitSHA,
			Created:       repo.Created,
			Updated:       repo.Updated,
		}
	}

	return c.OK(ListResponse{
		Repos: summaries,
		Total: result.Total,
	})
}
