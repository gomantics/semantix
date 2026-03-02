package repositories

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/indexing"
	"github.com/gomantics/semantix/internal/domains/repos"
	"go.uber.org/zap"
)

func Reindex(c web.Context) error {
	_, err := strconv.ParseInt(c.Param("wid"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	rid, err := strconv.ParseInt(c.Param("rid"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid repo id")
	}

	ctx := c.Request().Context()

	repo, err := repos.UpdateStatus(ctx, rid, repos.StatusPending, nil)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return c.NotFound("repo not found")
		}
		c.L.Error("failed to trigger reindex", zap.Error(err), zap.Int64("rid", rid))
		return c.InternalError("failed to trigger reindex")
	}

	indexing.Trigger()
	return c.OK(repo)
}
