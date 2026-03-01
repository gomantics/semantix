package repositories

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Configure(e *echo.Echo, l *zap.Logger) {
	e.GET("/v1/workspaces/:wid/repos", web.WrapAuth(List, l))
	e.POST("/v1/workspaces/:wid/repos", web.WrapAuth(Create, l))
	e.GET("/v1/workspaces/:wid/repos/:rid", web.WrapAuth(Get, l))
	e.DELETE("/v1/workspaces/:wid/repos/:rid", web.WrapAuth(Delete, l))
	e.POST("/v1/workspaces/:wid/repos/:rid/reindex", web.WrapAuth(Reindex, l))
	e.POST("/v1/repos/connectivity", web.WrapAuth(TestConnectivity, l))
}
