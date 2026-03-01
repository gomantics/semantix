package gittokens

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/gittokens"
	"go.uber.org/zap"
)

type UpdateRequest struct {
	Name  string `json:"name"`
	Token string `json:"token"`
}

func Update(c web.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid id")
	}

	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.Name == "" {
		return c.BadRequest("name is required")
	}
	if req.Token == "" {
		return c.BadRequest("token is required")
	}

	ctx := c.Request().Context()

	token, err := gittokens.Update(ctx, id, gittokens.UpdateParams{
		Name:  req.Name,
		Token: req.Token,
	})
	if err != nil {
		if errors.Is(err, gittokens.ErrNotFound) {
			return c.NotFound("git token not found")
		}
		c.L.Error("failed to update git token", zap.Error(err), zap.Int64("id", id))
		return c.InternalError("failed to update git token")
	}

	return c.OK(token)
}
