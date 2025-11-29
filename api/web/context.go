package web

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Context wraps echo.Context with additional fields
type Context struct {
	echo.Context
	L *zap.Logger
}

// HandlerFunc is a handler function that uses our custom Context
type HandlerFunc func(ctx Context) error

// Wrap wraps a handler function to use our custom context
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

// Error sends an error response
func (c Context) Error(status int, message string) error {
	return c.JSON(status, map[string]string{
		"error": message,
	})
}

// BadRequest sends a 400 error
func (c Context) BadRequest(message string) error {
	return c.Error(http.StatusBadRequest, message)
}

// NotFound sends a 404 error
func (c Context) NotFound(message string) error {
	return c.Error(http.StatusNotFound, message)
}

// InternalError sends a 500 error
func (c Context) InternalError(message string) error {
	return c.Error(http.StatusInternalServerError, message)
}

// OK sends a 200 response with data
func (c Context) OK(data any) error {
	return c.JSON(http.StatusOK, data)
}

// Created sends a 201 response with data
func (c Context) Created(data any) error {
	return c.JSON(http.StatusCreated, data)
}

// NoContent sends a 204 response
func (c Context) NoContent() error {
	return c.Context.NoContent(http.StatusNoContent)
}
