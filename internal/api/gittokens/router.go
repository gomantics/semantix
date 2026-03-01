package gittokens

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Configure(e *echo.Echo, l *zap.Logger) {
	e.GET("/v1/gittokens", web.WrapAuth(List, l))
	e.POST("/v1/gittokens", web.WrapAuth(Create, l))
	e.DELETE("/v1/gittokens/:id", web.WrapAuth(Delete, l))
}
