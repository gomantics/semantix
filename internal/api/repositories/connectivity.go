package repositories

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/gittokens"
	"github.com/gomantics/semantix/internal/libs/gitrepo"
	"go.uber.org/zap"
)

type ConnectivityRequest struct {
	URL        string `json:"url"`
	GitTokenID *int64 `json:"git_token_id,omitempty"`
}

type ConnectivityResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

func TestConnectivity(c web.Context) error {
	var req ConnectivityRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.URL == "" {
		return c.BadRequest("url is required")
	}

	ctx := c.Request().Context()
	provider, err := gitrepo.DetectProvider(req.URL)
	if err != nil {
		return c.BadRequest("unsupported git host: must be github.com or gitlab.com")
	}

	var token string
	if req.GitTokenID != nil {
		gt, err := gittokens.GetByID(ctx, *req.GitTokenID)
		if err != nil {
			return c.BadRequest("git token not found")
		}
		token = gt.Token
	}

	err = gitrepo.CheckConnectivity(ctx, gitrepo.CloneOptions{
		URL:      req.URL,
		Provider: provider,
		Token:    token,
	})
	if err != nil {
		c.L.Info("connectivity check failed", zap.String("url", req.URL), zap.Error(err))
		return c.JSON(200, ConnectivityResponse{OK: false, Message: "could not reach repository - check the URL and token"})
	}

	return c.JSON(200, ConnectivityResponse{OK: true})
}
