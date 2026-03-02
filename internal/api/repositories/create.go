package repositories

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/indexing"
	"github.com/gomantics/semantix/internal/domains/repos"
	"go.uber.org/zap"
)

type CreateRequest struct {
	URL        string `json:"url"`
	Branch     string `json:"branch,omitempty"`
	IsPrivate  bool   `json:"is_private,omitempty"`
	GitTokenID *int64 `json:"git_token_id,omitempty"`
}

func Create(c web.Context) error {
	wid, err := strconv.ParseInt(c.Param("wid"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.URL == "" {
		return c.BadRequest("url is required")
	}

	ctx := c.Request().Context()

	repo, err := repos.Create(ctx, repos.CreateParams{
		WorkspaceID: wid,
		GitTokenID:  req.GitTokenID,
		URL:         req.URL,
		Branch:      req.Branch,
		IsPrivate:   req.IsPrivate,
	})
	if err != nil {
		if errors.Is(err, repos.ErrTokenRequired) {
			return c.BadRequest("git_token_id is required for private repositories")
		}
		c.L.Error("failed to create repo", zap.Error(err), zap.Int64("wid", wid))
		return c.InternalError("failed to create repo")
	}

	indexing.Trigger()

	return c.Created(repo)
}
