package gittokens

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/gittokens"
	"github.com/gomantics/semantix/internal/libs/gitrepo"
	"go.uber.org/zap"
)

const defaultPerPage = 30
const maxPerPage = 100

type RemoteReposResponse struct {
	Repos      []gitrepo.RemoteRepo `json:"repos"`
	NextCursor string               `json:"next_cursor,omitempty"`
	PerPage    int                  `json:"per_page"`
}

func ListRemoteRepos(c web.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return c.BadRequest("invalid token id")
	}

	cursor := c.QueryParam("cursor")
	search := c.QueryParam("search")

	perPage := defaultPerPage
	if pp := c.QueryParam("per_page"); pp != "" {
		perPage, err = strconv.Atoi(pp)
		if err != nil || perPage < 1 {
			return c.BadRequest("invalid per_page")
		}
		if perPage > maxPerPage {
			perPage = maxPerPage
		}
	}

	ctx := c.Request().Context()

	token, err := gittokens.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gittokens.ErrNotFound) {
			return c.NotFound("git token not found")
		}
		c.L.Error("failed to get git token", zap.Error(err))
		return c.InternalError("failed to get git token")
	}

	page, err := gitrepo.ListRemoteReposPage(ctx, token.Provider, token.Token, cursor, perPage, search)
	if err != nil {
		c.L.Error("failed to list remote repos", zap.Error(err), zap.String("provider", string(token.Provider)))

		if strings.Contains(err.Error(), "invalid cursor") {
			return c.BadRequest("invalid cursor")
		}

		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			(errors.As(err, &netErr) && netErr.Timeout()) {
			return c.Error(http.StatusGatewayTimeout, "upstream provider timed out")
		}
		return c.Error(http.StatusBadGateway, "failed to reach git provider")
	}

	repos := page.Repos
	if repos == nil {
		repos = []gitrepo.RemoteRepo{}
	}

	return c.OK(RemoteReposResponse{
		Repos:      repos,
		NextCursor: page.NextCursor,
		PerPage:    perPage,
	})
}
