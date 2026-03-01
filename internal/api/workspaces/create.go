package workspaces

import (
	"errors"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/workspaces"
	"go.uber.org/zap"
)

type CreateRequest struct {
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Description *string                `json:"description,omitempty"`
	Settings    *workspaces.WorkspaceSettings `json:"settings,omitempty"`
}

func Create(c web.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.Name == "" {
		return c.BadRequest("name is required")
	}
	if req.Slug == "" {
		return c.BadRequest("slug is required")
	}

	ctx := c.Request().Context()

	ws, err := workspaces.Create(ctx, workspaces.CreateParams{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Settings:    req.Settings,
	})
	if err != nil {
		if errors.Is(err, workspaces.ErrAlreadyExists) {
			return c.BadRequest("workspace with this slug already exists")
		}
		c.L.Error("failed to create workspace", zap.Error(err))
		return c.InternalError("failed to create workspace")
	}

	return c.Created(ws)
}
