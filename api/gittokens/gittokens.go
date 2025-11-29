package gittokens

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/api/web"
	"github.com/gomantics/semantix/domains/gittokens"
	"go.uber.org/zap"
)

// CreateRequest is the request body for creating a git token
type CreateRequest struct {
	WorkspaceID int64  `json:"workspace_id"`
	Provider    string `json:"provider"`
	Name        string `json:"name"`
	Token       string `json:"token"`
}

// GitTokenResponse is the response for git token operations
// Note: We never return the actual token in responses
type GitTokenResponse struct {
	ID          int64  `json:"id"`
	WorkspaceID int64  `json:"workspace_id"`
	Provider    string `json:"provider"`
	Name        string `json:"name"`
	Created     int64  `json:"created"`
	Updated     int64  `json:"updated"`
}

// ListResponse is the response for listing git tokens
type ListResponse struct {
	Tokens []GitTokenResponse `json:"tokens"`
}

// Create handles POST /v1/git-tokens
func Create(c web.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.WorkspaceID <= 0 {
		return c.BadRequest("workspace_id is required")
	}
	if req.Provider == "" {
		return c.BadRequest("provider is required")
	}
	if req.Name == "" {
		return c.BadRequest("name is required")
	}
	if req.Token == "" {
		return c.BadRequest("token is required")
	}

	// Validate provider
	provider := gittokens.Provider(req.Provider)
	if provider != gittokens.ProviderGitHub && provider != gittokens.ProviderGitLab {
		return c.BadRequest("invalid provider, must be 'github' or 'gitlab'")
	}

	ctx := c.Request().Context()

	token, err := gittokens.Create(ctx, gittokens.CreateParams{
		WorkspaceID: req.WorkspaceID,
		Provider:    provider,
		Name:        req.Name,
		Token:       req.Token,
	})
	if errors.Is(err, gittokens.ErrAlreadyExists) {
		return c.BadRequest("git token with this name already exists for this provider")
	}
	if err != nil {
		c.L.Error("failed to create git token", zap.Error(err))
		return c.InternalError("failed to create git token")
	}

	c.L.Info("git token created",
		zap.Int64("id", token.ID),
		zap.Int64("workspace_id", token.WorkspaceID),
		zap.String("provider", string(token.Provider)),
	)

	return c.Created(GitTokenResponse{
		ID:          token.ID,
		WorkspaceID: token.WorkspaceID,
		Provider:    string(token.Provider),
		Name:        token.Name,
		Created:     token.Created,
		Updated:     token.Updated,
	})
}

// List handles GET /v1/git-tokens
func List(c web.Context) error {
	ctx := c.Request().Context()

	workspaceID, err := strconv.ParseInt(c.QueryParam("workspace_id"), 10, 64)
	if err != nil || workspaceID <= 0 {
		return c.BadRequest("workspace_id query parameter is required")
	}

	params := gittokens.ListParams{
		WorkspaceID: workspaceID,
	}

	if provider := c.QueryParam("provider"); provider != "" {
		p := gittokens.Provider(provider)
		params.Provider = &p
	}

	tokens, err := gittokens.ListByWorkspace(ctx, params)
	if err != nil {
		c.L.Error("failed to list git tokens", zap.Error(err))
		return c.InternalError("failed to list git tokens")
	}

	responses := make([]GitTokenResponse, len(tokens))
	for i, token := range tokens {
		responses[i] = GitTokenResponse{
			ID:          token.ID,
			WorkspaceID: token.WorkspaceID,
			Provider:    string(token.Provider),
			Name:        token.Name,
			Created:     token.Created,
			Updated:     token.Updated,
		}
	}

	return c.OK(ListResponse{Tokens: responses})
}

// Get handles GET /v1/git-tokens/:id
func Get(c web.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid git token id")
	}

	token, err := gittokens.GetByID(ctx, id)
	if errors.Is(err, gittokens.ErrNotFound) {
		return c.NotFound("git token not found")
	}
	if err != nil {
		c.L.Error("failed to get git token", zap.Error(err))
		return c.InternalError("failed to get git token")
	}

	return c.OK(GitTokenResponse{
		ID:          token.ID,
		WorkspaceID: token.WorkspaceID,
		Provider:    string(token.Provider),
		Name:        token.Name,
		Created:     token.Created,
		Updated:     token.Updated,
	})
}

// UpdateRequest is the request body for updating a git token
type UpdateRequest struct {
	Name  string `json:"name"`
	Token string `json:"token,omitempty"` // Optional - if empty, token is not updated
}

// Update handles PUT /v1/git-tokens/:id
func Update(c web.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid git token id")
	}

	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.Name == "" {
		return c.BadRequest("name is required")
	}

	token, err := gittokens.Update(ctx, id, gittokens.UpdateParams{
		Name:  req.Name,
		Token: req.Token,
	})
	if errors.Is(err, gittokens.ErrNotFound) {
		return c.NotFound("git token not found")
	}
	if err != nil {
		c.L.Error("failed to update git token", zap.Error(err))
		return c.InternalError("failed to update git token")
	}

	return c.OK(GitTokenResponse{
		ID:          token.ID,
		WorkspaceID: token.WorkspaceID,
		Provider:    string(token.Provider),
		Name:        token.Name,
		Created:     token.Created,
		Updated:     token.Updated,
	})
}

// Delete handles DELETE /v1/git-tokens/:id
func Delete(c web.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid git token id")
	}

	if err := gittokens.Delete(ctx, id); err != nil {
		if errors.Is(err, gittokens.ErrNotFound) {
			return c.NotFound("git token not found")
		}
		c.L.Error("failed to delete git token", zap.Error(err))
		return c.InternalError("failed to delete git token")
	}

	c.L.Info("git token deleted", zap.Int64("id", id))
	return c.NoContent()
}
