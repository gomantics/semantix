package auth

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func Configure(e *echo.Echo, l *zap.Logger) {
	e.POST("/v1/auth/signup", web.Wrap(Signup, l))
	e.POST("/v1/auth/login", web.Wrap(Login, l))
	e.POST("/v1/auth/logout", web.Wrap(Logout, l))
	e.GET("/v1/auth/me", web.WrapAuth(Me, l))
}
