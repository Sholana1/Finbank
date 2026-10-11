package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/sholana1/fintechdemo/pkg/pgdb"

	_ "github.com/jackc/pgx/v5/stdlib"

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

	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	applied, err := pgdb.Migrate(ctx, url, migrations.FS)
	if err != nil {
		return err
	}

	logger.Info("schema up to date", "applied", applied)

	return nil
}
