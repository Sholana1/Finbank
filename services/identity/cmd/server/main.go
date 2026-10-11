package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	identityv1 "github.com/sholana1/fintechdemo/gen/identity/v1"
	"github.com/sholana1/fintechdemo/identity/internal/adapters/grpcapi"
	"github.com/sholana1/fintechdemo/identity/internal/adapters/passwordhash"
	"github.com/sholana1/fintechdemo/identity/internal/adapters/postgres"
	"github.com/sholana1/fintechdemo/identity/internal/app"
	"github.com/sholana1/fintechdemo/identity/internal/config"
	"github.com/sholana1/fintechdemo/pkg/grpcserver"
	"github.com/sholana1/fintechdemo/pkg/pgdb"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger = logger.With("service", cfg.ServiceName, "env", cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgdb.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	logger.Info("database pool ready")

	users := postgres.NewUserRepository(pool)
	hasher := passwordhash.NewArgon2id(passwordhash.DefaultParams)
	registerUser := app.NewRegisterUser(users, hasher, utcNow)
	handler := grpcapi.NewHandler(registerUser, logger)

	server := grpcserver.NewServer(logger, pool)
	identityv1.RegisterIdentityServiceServer(server, handler)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return err
	}

	return server.Run(ctx, lis)

}

func utcNow() time.Time {
	return time.Now().UTC()
}
