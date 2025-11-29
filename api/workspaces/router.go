package workspaces

import (
	"github.com/gomantics/semantix/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Configure(e *echo.Echo, l *zap.Logger) {
	e.POST("/v1/workspaces", web.Wrap(Create, l))
	e.GET("/v1/workspaces", web.Wrap(List, l))
	e.GET("/v1/workspaces/:id", web.Wrap(Get, l))
	e.PUT("/v1/workspaces/:id", web.Wrap(Update, l))
	e.DELETE("/v1/workspaces/:id", web.Wrap(Delete, l))
}
