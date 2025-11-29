package workspaces

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/api/web"
	"github.com/gomantics/semantix/domains/workspaces"
	"go.uber.org/zap"
)

// WorkspaceResponse is the response for workspace operations
type WorkspaceResponse struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
}

// ListResponse is the response for listing workspaces
type ListResponse struct {
	Workspaces []WorkspaceResponse `json:"workspaces"`
	Total      int64               `json:"total"`
}

// Create handles POST /v1/workspaces
func Create(c web.Context) error {
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}

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
		Name: req.Name,
		Slug: req.Slug,
	})
	if errors.Is(err, workspaces.ErrAlreadyExists) {
		return c.BadRequest("workspace with this slug already exists")
	}
	if err != nil {
		c.L.Error("failed to create workspace", zap.Error(err))
		return c.InternalError("failed to create workspace")
	}

	c.L.Info("workspace created",
		zap.Int64("id", ws.ID),
		zap.String("slug", ws.Slug),
	)

	return c.Created(map[string]any{
		"workspace": WorkspaceResponse{
			ID:      ws.ID,
			Name:    ws.Name,
			Slug:    ws.Slug,
			Created: ws.Created,
			Updated: ws.Updated,
		},
	})
}

// List handles GET /v1/workspaces
func List(c web.Context) error {
	ctx := c.Request().Context()

	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	result, err := workspaces.List(ctx, workspaces.ListParams{
		Limit:  limit,
		Offset: (page - 1) * limit,
	})
	if err != nil {
		c.L.Error("failed to list workspaces", zap.Error(err))
		return c.InternalError("failed to list workspaces")
	}

	responses := make([]WorkspaceResponse, len(result.Workspaces))
	for i, ws := range result.Workspaces {
		responses[i] = WorkspaceResponse{
			ID:      ws.ID,
			Name:    ws.Name,
			Slug:    ws.Slug,
			Created: ws.Created,
			Updated: ws.Updated,
		}
	}

	return c.OK(ListResponse{
		Workspaces: responses,
		Total:      result.Total,
	})
}

// Get handles GET /v1/workspaces/:id
func Get(c web.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	ws, err := workspaces.GetByID(ctx, id)
	if errors.Is(err, workspaces.ErrNotFound) {
		return c.NotFound("workspace not found")
	}
	if err != nil {
		c.L.Error("failed to get workspace", zap.Error(err))
		return c.InternalError("failed to get workspace")
	}

	return c.OK(map[string]any{
		"workspace": WorkspaceResponse{
			ID:      ws.ID,
			Name:    ws.Name,
			Slug:    ws.Slug,
			Created: ws.Created,
			Updated: ws.Updated,
		},
	})
}

// UpdateRequest is the request body for updating a workspace
type UpdateRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Update handles PUT /v1/workspaces/:id
func Update(c web.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.Name == "" {
		return c.BadRequest("name is required")
	}
	if req.Slug == "" {
		return c.BadRequest("slug is required")
	}

	ws, err := workspaces.Update(ctx, id, workspaces.UpdateParams{
		Name: req.Name,
		Slug: req.Slug,
	})
	if errors.Is(err, workspaces.ErrNotFound) {
		return c.NotFound("workspace not found")
	}
	if errors.Is(err, workspaces.ErrAlreadyExists) {
		return c.BadRequest("workspace with this slug already exists")
	}
	if err != nil {
		c.L.Error("failed to update workspace", zap.Error(err))
		return c.InternalError("failed to update workspace")
	}

	return c.OK(map[string]any{
		"workspace": WorkspaceResponse{
			ID:      ws.ID,
			Name:    ws.Name,
			Slug:    ws.Slug,
			Created: ws.Created,
			Updated: ws.Updated,
		},
	})
}

// Delete handles DELETE /v1/workspaces/:id
func Delete(c web.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	if err := workspaces.Delete(ctx, id); err != nil {
		if errors.Is(err, workspaces.ErrNotFound) {
			return c.NotFound("workspace not found")
		}
		c.L.Error("failed to delete workspace", zap.Error(err))
		return c.InternalError("failed to delete workspace")
	}

	c.L.Info("workspace deleted", zap.Int64("id", id))
	return c.NoContent()
}
