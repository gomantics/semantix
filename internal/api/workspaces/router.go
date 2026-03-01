package workspaces

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Configure(e *echo.Echo, l *zap.Logger) {
	e.GET("/v1/workspaces", web.WrapAuth(List, l))
	e.POST("/v1/workspaces", web.WrapAuth(Create, l))
	e.GET("/v1/workspaces/:wid", web.WrapAuth(Get, l))
	e.DELETE("/v1/workspaces/:wid", web.WrapAuth(Delete, l))
}
