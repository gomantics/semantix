package workspaces

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/workspaces"
	"go.uber.org/zap"
)

func Delete(c web.Context) error {
	wid, err := strconv.ParseInt(c.Param("wid"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	ctx := c.Request().Context()

	if err := workspaces.Delete(ctx, wid); err != nil {
		if errors.Is(err, workspaces.ErrNotFound) {
			return c.NotFound("workspace not found")
		}
		c.L.Error("failed to delete workspace", zap.Error(err), zap.Int64("wid", wid))
		return c.InternalError("failed to delete workspace")
	}

	return c.NoContent()
}
