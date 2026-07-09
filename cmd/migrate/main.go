// Command migrate applies the goose database migrations embedded in the
// migrations package.
//
// Usage:
//
//	go run ./cmd/migrate up       # apply all pending migrations (default)
//	go run ./cmd/migrate down     # roll back the last migration
//	go run ./cmd/migrate status   # print migration status
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"

	"github.com/shseooo/go-architecture/internal/platform/database"
	"github.com/shseooo/go-architecture/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	_ = godotenv.Load()
	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("mysql"); err != nil {
		return err
	}
	return goose.RunContext(context.Background(), command, db, ".")
}
