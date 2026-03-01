package repositories

import (
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/repos"
	"go.uber.org/zap"
)

type ListResponse struct {
	Repos []repos.Repo `json:"repos"`
	Total int64        `json:"total"`
}

func List(c web.Context) error {
	wid, err := strconv.ParseInt(c.Param("wid"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid workspace id")
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	ctx := c.Request().Context()

	result, err := repos.List(ctx, repos.ListParams{
		WorkspaceID: wid,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		c.L.Error("failed to list repos", zap.Error(err), zap.Int64("wid", wid))
		return c.InternalError("failed to list repos")
	}

	return c.OK(ListResponse{
		Repos: result.Repos,
		Total: result.Total,
	})
}
