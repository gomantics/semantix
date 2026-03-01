package repositories

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/repos"
	"go.uber.org/zap"
)

func Get(c web.Context) error {
	_, err := strconv.ParseInt(c.Param("wid"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	rid, err := strconv.ParseInt(c.Param("rid"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid repo id")
	}

	ctx := c.Request().Context()

	repo, err := repos.GetByID(ctx, rid)
	if err != nil {
		if errors.Is(err, repos.ErrNotFound) {
			return c.NotFound("repo not found")
		}
		c.L.Error("failed to get repo", zap.Error(err), zap.Int64("rid", rid))
		return c.InternalError("failed to get repo")
	}

	return c.OK(repo)
}
