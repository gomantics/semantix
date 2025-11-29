package repositories

import (
	"github.com/gomantics/semantix/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Configure(e *echo.Echo, l *zap.Logger) {
	e.POST("/v1/repositories", web.Wrap(Create, l))
	e.GET("/v1/repositories", web.Wrap(List, l))
	e.GET("/v1/repositories/:id", web.Wrap(Get, l))
	e.POST("/v1/repositories/:id/reindex", web.Wrap(Reindex, l))
	e.DELETE("/v1/repositories/:id", web.Wrap(Delete, l))
}
