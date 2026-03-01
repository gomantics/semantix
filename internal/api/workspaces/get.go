package workspaces

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/workspaces"
	"go.uber.org/zap"
)

func Get(c web.Context) error {
	wid, err := strconv.ParseInt(c.Param("wid"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	ctx := c.Request().Context()

	ws, err := workspaces.GetByID(ctx, wid)
	if err != nil {
		if errors.Is(err, workspaces.ErrNotFound) {
			return c.NotFound("workspace not found")
		}
		c.L.Error("failed to get workspace", zap.Error(err), zap.Int64("wid", wid))
		return c.InternalError("failed to get workspace")
	}

	return c.OK(ws)
}
