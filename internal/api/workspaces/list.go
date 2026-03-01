package workspaces

import (
	"strconv"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/workspaces"
	"go.uber.org/zap"
)

type ListResponse struct {
	Workspaces []workspaces.Workspace `json:"workspaces"`
	Total      int64                  `json:"total"`
}

func List(c web.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	ctx := c.Request().Context()

	result, err := workspaces.List(ctx, workspaces.ListParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		c.L.Error("failed to list workspaces", zap.Error(err))
		return c.InternalError("failed to list workspaces")
	}

	return c.OK(ListResponse{
		Workspaces: result.Workspaces,
		Total:      result.Total,
	})
}
