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

	user, err := users.CreateFirst(ctx, users.CreateParams{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, users.ErrAdminExists) {
			return c.Error(http.StatusForbidden, "admin user already exists")
		}
		c.L.Error("failed to create user", zap.Error(err))
		return c.InternalError("failed to create user")
	}

	token, err := users.CreateSession(ctx, user.ID)
	if err != nil {
		c.L.Error("failed to create session", zap.Error(err))
		return c.InternalError("failed to create session")
	}

	c.SetCookie(sessionCookie(token))

	return c.Created(map[string]any{
		"id":    user.ID,
		"email": user.Email,
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
