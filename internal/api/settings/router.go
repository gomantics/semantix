package settings

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Configure(e *echo.Echo, l *zap.Logger) {
	e.GET("/v1/settings/status", web.Wrap(Status, l))
	e.GET("/v1/settings", web.WrapAuth(Get, l))
	e.PUT("/v1/settings", web.WrapAuth(Update, l))
}
