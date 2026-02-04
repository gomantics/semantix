package web

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type Context struct {
	echo.Context
	L *zap.Logger
}

type HandlerFunc func(ctx Context) error

func Wrap(h HandlerFunc, l *zap.Logger) echo.HandlerFunc {
	return func(c echo.Context) error {
		rid := c.Response().Header().Get(echo.HeaderXRequestID)

		ctx := Context{
			Context: c,
			L:       l.With(zap.String("request_id", rid)),
		}

		return h(ctx)
	}
}

func (c Context) Error(status int, message string) error {
	return c.JSON(status, map[string]string{
		"error": message,
	})
}

func (c Context) BadRequest(message string) error {
	return c.Error(http.StatusBadRequest, message)
}

func (c Context) NotFound(message string) error {
	return c.Error(http.StatusNotFound, message)
}

func (c Context) InternalError(message string) error {
	return c.Error(http.StatusInternalServerError, message)
}

func (c Context) OK(data any) error {
	return c.JSON(http.StatusOK, data)
}

func (c Context) Created(data any) error {
	return c.JSON(http.StatusCreated, data)
}

func (c Context) NoContent() error {
	return c.Context.NoContent(http.StatusNoContent)
}
