package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/gomantics/semantix/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

//go:embed schema/*.sql
var embedSchema embed.FS

var defaultPool *pgxpool.Pool

// Init initializes the database connection pool and applies schema
func Init(lc fx.Lifecycle, l *zap.Logger) error {
	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(config.Database.Dsn())
	if err != nil {
		return fmt.Errorf("failed to parse database config: %w", err)
	}

	// Connection pool settings
	poolConfig.MaxConns = 50
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	defaultPool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
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

	// Apply schema
	if err := ApplySchema(ctx, l); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	l.Info("database schema applied")
	return nil
}

// GetPool returns the default connection pool
func GetPool() *pgxpool.Pool {
	return defaultPool
}

// Query runs fn with a Queries instance (can add retries here later)
func Query(ctx context.Context, fn func(*Queries) error) error {
	// TODO: add retry logic here
	return fn(New(defaultPool))
}

// Query1 runs fn and returns a single result
func Query1[T any](ctx context.Context, fn func(*Queries) (T, error)) (T, error) {
	// TODO: add retry logic here
	return fn(New(defaultPool))
}

// Tx runs fn within a transaction (can add retries here later)
func Tx(ctx context.Context, fn func(*Queries) error) error {
	// TODO: add retry logic here
	tx, err := defaultPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(New(tx)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Tx1 runs fn within a transaction and returns a result
func Tx1[T any](ctx context.Context, fn func(*Queries) (T, error)) (T, error) {
	// TODO: add retry logic here
	var zero T
	tx, err := defaultPool.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer tx.Rollback(ctx)

	result, err := fn(New(tx))
	if err != nil {
		return zero, err
	}

	if err := tx.Commit(ctx); err != nil {
		return zero, err
	}
	return result, nil
}

// ApplySchema applies all SQL schema files
func ApplySchema(ctx context.Context, l *zap.Logger) error {
	if defaultPool == nil {
		return fmt.Errorf("pool not initialized")
	}

	// Get all SQL files from embedded schema
	sqlFiles, err := getSchemaSQLFiles()
	if err != nil {
		return err
	}

	l.Info("found schema files", zap.Strings("files", sqlFiles))

	// Acquire a connection for schema operations
	conn, err := defaultPool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Execute each SQL file
	for _, filename := range sqlFiles {
		filePath := "schema/" + filename

		l.Info("executing schema file", zap.String("file", filename))

		content, err := embedSchema.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read schema file %s: %w", filename, err)
		}

		_, err = conn.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to execute schema %s: %w", filename, err)
		}

		l.Info("schema file executed", zap.String("file", filename))
	}

	return nil
}

// getSchemaSQLFiles returns sorted list of SQL files from embedded schema
func getSchemaSQLFiles() ([]string, error) {
	fsys, err := fs.Sub(embedSchema, "schema")
	if err != nil {
		return nil, err
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}

	var sqlFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}

	sort.Strings(sqlFiles)
	return sqlFiles, nil
}
