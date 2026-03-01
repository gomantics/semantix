package auth

import (
	"net/http"
	"time"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/users"
	"go.uber.org/zap"
)

func Logout(c web.Context) error {
	cookie, err := c.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		return c.NoContent()
	}

	ctx := c.Request().Context()
	if err := users.DeleteSession(ctx, cookie.Value); err != nil {
		c.L.Error("failed to delete session", zap.Error(err))
	}

	c.SetCookie(&http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	return c.NoContent()
}
