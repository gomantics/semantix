package health

import (
	"github.com/gomantics/semantix/api/web"
	"github.com/gomantics/semantix/db"
)

// GetResponse is the health check response
type GetResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// Get handles GET /v1/health
func Get(c web.Context) error {
	ctx := c.Request().Context()

	// Check database
	dbStatus := "ok"
	err := db.Query(ctx, func(q *db.Queries) error {
		return nil // just checking connection
	})
	if err != nil {
		dbStatus = "error: " + err.Error()
	}

	return c.OK(GetResponse{
		Status:   "ok",
		Database: dbStatus,
	})
}
