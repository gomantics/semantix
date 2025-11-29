package repositories

import (
	"errors"
	"strconv"

	"github.com/gomantics/semantix/api/web"
	"github.com/gomantics/semantix/domains/repos"
	"github.com/gomantics/semantix/libs/gitrepo"
	"github.com/gomantics/semantix/libs/milvus"
	"go.uber.org/zap"
)

// DeleteResponse is the response for deleting a repository
type DeleteResponse struct {
	Message string `json:"message"`
}

// Delete handles DELETE /v1/repositories/:id
func Delete(c web.Context) error {
	ctx := c.Request().Context()

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid repository id")
	}

	// Delete from Milvus first
	if err := milvus.DeleteByRepoID(ctx, id); err != nil {
		c.L.Warn("failed to delete from milvus", zap.Error(err))
	}

	// Delete from database
	err = repos.Delete(ctx, id)
	if errors.Is(err, repos.ErrNotFound) {
		return c.NotFound("repository not found")
	}
	if err != nil {
		c.L.Error("failed to delete repo", zap.Error(err))
		return c.InternalError("failed to delete repository")
	}

	// Cleanup cloned directory
	repoPath := gitrepo.GetRepoPath(id)
	if err := gitrepo.CleanupRepo(repoPath); err != nil {
		c.L.Warn("failed to cleanup repo directory", zap.Error(err))
	}

	c.L.Info("repository deleted", zap.Int64("repo_id", id))

	return c.OK(DeleteResponse{
		Message: "Repository deleted successfully",
	})
}
