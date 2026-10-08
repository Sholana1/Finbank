package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/sholana1/fintechdemo/ledger/internal/config"
	"github.com/sholana1/fintechdemo/ledger/migrations"
)

const migrationTimeout = 2 * time.Minute

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "ledger-migrate")

	if err := run(logger); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	url, err := config.MigrationDatabaseURL()
	if err != nil {
		return err
	}

	db, err := sql.Open("pgx", url)
	if err != nil {
		return err

	}

	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	results, err := provider.Up(ctx)
	if err != nil {
		return nil
	}

	for _, r := range results {
		logger.Info("migration applied", "file", r.Source.Path, "duration", r.Duration.String())
	}

	logger.Info("schema up to date", "applied", len(results))

	return nil
}
