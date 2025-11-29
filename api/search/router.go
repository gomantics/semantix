package search

import (
	"github.com/gomantics/semantix/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Configure sets up the search routes
func Configure(e *echo.Echo, l *zap.Logger) {
	e.POST("/v1/search", web.Wrap(Search, l))
}
