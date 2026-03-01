package auth

import (
	"github.com/gomantics/semantix/internal/api/web"
)

func Me(c web.Context) error {
	return c.OK(map[string]any{
		"id":    c.UserID,
		"email": c.UserEmail,
	})
}
