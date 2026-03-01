package auth

import (
	"errors"
	"net/http"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/users"
	"go.uber.org/zap"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(c web.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return c.BadRequest("email and password are required")
	}

	ctx := c.Request().Context()

	user, err := users.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, users.ErrInvalidCreds) {
			return c.Error(http.StatusUnauthorized, "invalid email or password")
		}
		c.L.Error("failed to login", zap.Error(err))
		return c.InternalError("failed to login")
	}

	token, err := users.CreateSession(ctx, user.ID)
	if err != nil {
		c.L.Error("failed to create session", zap.Error(err))
		return c.InternalError("failed to create session")
	}

	c.SetCookie(sessionCookie(token))

	return c.OK(map[string]any{
		"id":    user.ID,
		"email": user.Email,
	})
}
