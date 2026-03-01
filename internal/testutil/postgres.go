package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/gomantics/semantix/internal/db"
	"github.com/gomantics/semantix/internal/db/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver for goose
	"github.com/pressly/goose/v3"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// WithPostgres returns an Option that starts a Postgres 17.7 container, runs
// migrations, and injects the pool into the db package. The container is
// terminated on teardown.
func WithPostgres() Option {
	return func() Teardown {
		return withPostgres()
	}
}

func withPostgres() Teardown {
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:17.7-alpine",
		tcpostgres.WithDatabase("semantix"),
		tcpostgres.WithUsername("semantix"),
		tcpostgres.WithPassword("semantix"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("testutil: start postgres: %v", err))
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(fmt.Sprintf("testutil: get postgres DSN: %v", err))
	}

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		panic(fmt.Sprintf("testutil: parse pool config: %v", err))
	}
	poolCfg.MaxConns = 10
	poolCfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		panic(fmt.Sprintf("testutil: create pgxpool: %v", err))
	}

	if err := pool.Ping(ctx); err != nil {
		panic(fmt.Sprintf("testutil: ping postgres: %v", err))
	}

	if err := runMigrations(ctx, dsn); err != nil {
		panic(fmt.Sprintf("testutil: run migrations: %v", err))
	}

	db.SetPool(pool)

	return func() {
		pool.Close()
		if err := testcontainers.TerminateContainer(ctr); err != nil {
			fmt.Printf("testutil: terminate postgres: %v\n", err)
		}
	}
}

func runMigrations(ctx context.Context, dsn string) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	sqlDB, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer sqlDB.Close()

	if err := goose.RunContext(ctx, "up", sqlDB, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
