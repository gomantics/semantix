package auth

import (
	"errors"
	"net/http"

	"github.com/gomantics/semantix/config"
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/users"
	"go.uber.org/zap"
)

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Signup(c web.Context) error {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.Email == "" {
		return c.BadRequest("email is required")
	}
	if len(req.Password) < 8 {
		return c.BadRequest("password must be at least 8 characters")
	}

	ctx := c.Request().Context()

	result, err := users.Signup(ctx, users.CreateParams{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, users.ErrAdminExists) {
			return c.Error(http.StatusForbidden, "admin user already exists")
		}
		c.L.Error("failed to sign up", zap.Error(err))
		return c.InternalError("failed to sign up")
	}

	c.SetCookie(sessionCookie(result.Token))

	return c.Created(map[string]any{
		"id":    result.User.ID,
		"email": result.User.Email,
	})
}

func sessionCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   config.IsProd(),
	}
}
