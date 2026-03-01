package repositories

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Configure(e *echo.Echo, l *zap.Logger) {
	g := e.Group("/v1/workspaces/:wid/repos")
	g.GET("", web.WrapAuth(List, l))
	g.POST("", web.WrapAuth(Create, l))
	g.GET("/:rid", web.WrapAuth(Get, l))
	g.DELETE("/:rid", web.WrapAuth(Delete, l))
	g.POST("/:rid/reindex", web.WrapAuth(Reindex, l))
}
