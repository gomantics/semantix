// cmd/migrate runs database migrations for Semantix using goose.
//
// Usage:
//
//	go run ./cmd/migrate [command]
//	go run ./cmd/migrate up       (default when no command given)
//	go run ./cmd/migrate status
//	go run ./cmd/migrate version
//	go run ./cmd/migrate reset    (prompts for confirmation)
//
// The DSN is read from the CONFIG_DATABASE_DSN environment variable, falling
// back to the default from the config package (localhost dev database).
package main

import (
	"bufio"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gomantics/semantix/config"
	"github.com/gomantics/semantix/internal/db/migrations"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const migrationsDir = "."

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: migrate [command]\n\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  up      apply all pending migrations (default)\n")
		fmt.Fprintf(os.Stderr, "  status  show applied and pending migrations\n")
		fmt.Fprintf(os.Stderr, "  version print current migration version\n")
		fmt.Fprintf(os.Stderr, "  reset   drop all tables and re-run every migration (dev only, prompts for confirmation)\n")
	}
	flag.Parse()

	db, err := sql.Open("pgx", config.Database.Dsn())
	if err != nil {
		log.Fatalf("migrate: failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("migrate: failed to connect to database: %v", err)
	}

	goose.SetBaseFS(migrations.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("migrate: failed to set dialect: %v", err)
	}

	cmd := "up"
	if args := flag.Args(); len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "up":
		if err := goose.Up(db, migrationsDir); err != nil {
			log.Fatalf("migrate: up failed: %v", err)
		}
	case "status":
		if err := goose.Status(db, migrationsDir); err != nil {
			log.Fatalf("migrate: status failed: %v", err)
		}
	case "version":
		if err := goose.Version(db, migrationsDir); err != nil {
			log.Fatalf("migrate: version failed: %v", err)
		}
	case "reset":
		runReset(db)
	default:
		fmt.Fprintf(os.Stderr, "migrate: unknown command %q\n\n", cmd)
		flag.Usage()
		os.Exit(1)
	}
}

// runReset prompts for confirmation then rolls all migrations back to zero
// and re-applies them. Intended for local development only - destroys all data.
func runReset(db *sql.DB) {
	fmt.Print("This will drop all tables and re-run every migration. Type \"yes\" to continue: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.TrimSpace(scanner.Text()) != "yes" {
		fmt.Println("migrate: reset cancelled")
		os.Exit(0)
	}

	log.Println("migrate: resetting database - rolling back all migrations")
	if err := goose.DownTo(db, migrationsDir, 0); err != nil {
		log.Fatalf("migrate: reset (down-to 0) failed: %v", err)
	}

	log.Println("migrate: re-applying all migrations")
	if err := goose.Up(db, migrationsDir); err != nil {
		log.Fatalf("migrate: reset (up) failed: %v", err)
	}

	log.Println("migrate: reset complete")
}
