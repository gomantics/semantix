package db

import (
	"context"
	"fmt"
	"time"

	"github.com/gomantics/semantix/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var defaultPool *pgxpool.Pool

// Init initializes the database connection pool.
//
// Schema migrations are not run here. Run `go run ./cmd/migrate up` before
// starting the service to apply any pending migrations.
func Init(lc fx.Lifecycle, l *zap.Logger) error {
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(config.Database.Dsn())
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = 50
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	defaultPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := defaultPool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			l.Info("closing database pool")
			if defaultPool != nil {
				defaultPool.Close()
			}
			return nil
		},
	})

	l.Info("database pool initialized")
	return nil
}

// GetPool returns the default connection pool.
func GetPool() *pgxpool.Pool {
	return defaultPool
}
