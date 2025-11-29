package repositories

import (
	"errors"

	"github.com/gomantics/semantix/api/web"
	"github.com/gomantics/semantix/domains/repos"
	"github.com/gomantics/semantix/libs/gitrepo"
	"go.uber.org/zap"
)

// CreateRequest is the request body for creating a repository
type CreateRequest struct {
	WorkspaceID int64  `json:"workspace_id"`
	GitTokenID  *int64 `json:"git_token_id,omitempty"`
	URL         string `json:"url"`
}

// CreateResponse is the response for creating a repository
type CreateResponse struct {
	ID          int64  `json:"id"`
	WorkspaceID int64  `json:"workspace_id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
}

// Create handles POST /v1/repositories
func Create(c web.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.WorkspaceID <= 0 {
		return c.BadRequest("workspace_id is required")
	}

	if req.URL == "" {
		return c.BadRequest("url is required")
	}

	if err := gitrepo.ValidateRepoURL(req.URL); err != nil {
		return c.BadRequest(err.Error())
	}

	ctx := c.Request().Context()

	repo, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: req.WorkspaceID,
		GitTokenID:  req.GitTokenID,
		URL:         req.URL,
	})
	if errors.Is(err, repos.ErrAlreadyExists) {
		// Return existing repo
		return c.OK(CreateResponse{
			ID:          repo.ID,
			WorkspaceID: repo.WorkspaceID,
			URL:         repo.URL,
			Name:        repo.Name,
			Owner:       repo.Owner,
			Status:      repo.Status.String(),
		})
	}
	if err != nil {
		c.L.Error("failed to create repo", zap.Error(err))
		return c.InternalError("failed to create repository")
	}

	c.L.Info("repository created",
		zap.Int64("id", repo.ID),
		zap.Int64("workspace_id", repo.WorkspaceID),
		zap.String("url", req.URL),
	)

	return c.Created(CreateResponse{
		ID:          repo.ID,
		WorkspaceID: repo.WorkspaceID,
		URL:         repo.URL,
		Name:        repo.Name,
		Owner:       repo.Owner,
		Status:      repo.Status.String(),
	})
}
