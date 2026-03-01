package gittokens

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/gittokens"
	"go.uber.org/zap"
)

type CreateRequest struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Token    string `json:"token"`
}

func Create(c web.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.Name == "" {
		return c.BadRequest("name is required")
	}
	if req.Provider == "" {
		return c.BadRequest("provider is required")
	}
	if req.Token == "" {
		return c.BadRequest("token is required")
	}

	ctx := c.Request().Context()

	token, err := gittokens.Create(ctx, gittokens.CreateParams{
		Name:     req.Name,
		Provider: req.Provider,
		Token:    req.Token,
	})
	if err != nil {
		c.L.Error("failed to create git token", zap.Error(err))
		return c.InternalError("failed to create git token")
	}

	return c.Created(token)
}
