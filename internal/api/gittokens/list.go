package gittokens

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/gittokens"
	"go.uber.org/zap"
)

type ListResponse struct {
	Tokens []gittokens.GitToken `json:"tokens"`
}

func List(c web.Context) error {
	ctx := c.Request().Context()

	tokens, err := gittokens.List(ctx)
	if err != nil {
		c.L.Error("failed to list git tokens", zap.Error(err))
		return c.InternalError("failed to list git tokens")
	}

	return c.OK(ListResponse{Tokens: tokens})
}
