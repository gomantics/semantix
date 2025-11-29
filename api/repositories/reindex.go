package repositories

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/api/web"
	"github.com/gomantics/semantix/domains/repos"
	"go.uber.org/zap"
)

// ReindexResponse is the response for reindexing a repository
type ReindexResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Reindex handles POST /v1/repositories/:id/reindex
func Reindex(c web.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid repository id")
	}

	err = repos.TriggerReindex(ctx, id)
	if errors.Is(err, repos.ErrNotFound) {
		return c.NotFound("repository not found")
	}
	if errors.Is(err, repos.ErrAlreadyActive) {
		return c.BadRequest("repository is already being indexed")
	}
	if err != nil {
		c.L.Error("failed to trigger reindex", zap.Error(err))
		return c.InternalError("failed to trigger reindex")
	}

	c.L.Info("reindex triggered", zap.Int64("repo_id", id))

	return c.OK(ReindexResponse{
		Status:  repos.StatusPending.String(),
		Message: "Re-indexing has been queued",
	})
}
