package search

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Configure(e *echo.Echo, l *zap.Logger) {
	e.POST("/v1/workspaces/:wid/search", web.WrapAuth(Search, l))
}
