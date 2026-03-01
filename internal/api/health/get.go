package health

import (
	"net/http"

	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/qdrant"
)

// Version is the current application version
const Version = "0.1.0"

// GetResponse is the health check response
type GetResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Qdrant   string `json:"qdrant"`
	Version  string `json:"version"`
}

// Get handles GET /v1/health
func Get(c web.Context) error {
	ctx := c.Request().Context()
	overallStatus := "ok"

	// PostgreSQL: Use pool.Ping() for proper connectivity check
	dbStatus := "ok"
	if err := db.GetPool().Ping(ctx); err != nil {
		dbStatus = "error"
		overallStatus = "degraded"
	}

	// Qdrant: Check health via gRPC HealthCheck API
	qdrantStatus := "ok"
	if err := qdrant.HealthCheck(ctx); err != nil {
		qdrantStatus = "error"
		overallStatus = "degraded"
	}

	resp := GetResponse{
		Status:   overallStatus,
		Database: dbStatus,
		Qdrant:   qdrantStatus,
		Version:  Version,
	}

	if overallStatus != "ok" {
		return c.JSON(http.StatusServiceUnavailable, resp)
	}

	return c.OK(resp)
}
