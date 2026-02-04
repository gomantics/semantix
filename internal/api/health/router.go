package health

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Configure sets up the health routes
func Configure(e *echo.Echo, l *zap.Logger) {
	e.GET("/v1/health", web.Wrap(Get, l))
}
