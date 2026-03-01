package gittokens

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/gittokens"
	"go.uber.org/zap"
)

func Delete(c web.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid id")
	}

	ctx := c.Request().Context()

	if err := gittokens.Delete(ctx, id); err != nil {
		if errors.Is(err, gittokens.ErrNotFound) {
			return c.NotFound("git token not found")
		}
		c.L.Error("failed to delete git token", zap.Error(err), zap.Int64("id", id))
		return c.InternalError("failed to delete git token")
	}

	return c.NoContent()
}
