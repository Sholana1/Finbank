package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	identityv1 "github.com/sholana1/fintechdemo/gen/identity/v1"
	"github.com/sholana1/fintechdemo/identity/internal/adapters/accesstoken"
	"github.com/sholana1/fintechdemo/identity/internal/adapters/grpcapi"
	"github.com/sholana1/fintechdemo/identity/internal/adapters/passwordhash"
	"github.com/sholana1/fintechdemo/identity/internal/adapters/postgres"
	"github.com/sholana1/fintechdemo/identity/internal/adapters/redisstore"
	"github.com/sholana1/fintechdemo/identity/internal/app"
	"github.com/sholana1/fintechdemo/identity/internal/config"
	"github.com/sholana1/fintechdemo/pkg/grpcserver"
	"github.com/sholana1/fintechdemo/pkg/pgdb"
)

const startupTimeout = 10 * time.Second

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

	redisClient, err := newRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer redisClient.Close()
	logger.Info("redis ready")

	// Adapters.
	db := pgdb.NewDB(pool)
	users := postgres.NewUserRepository(db)
	sessions := postgres.NewSessionRepository(db)
	refreshTokens := postgres.NewRefreshTokenRepository(db)
	loginAttempts := postgres.NewLoginAttemptRepository(db)

	hasher := passwordhash.NewArgon2id(passwordhash.DefaultParams, cfg.PasswordHashConcurrency)
	limiter := redisstore.NewLoginLimiter(redisClient, cfg.LoginMaxFailures, cfg.LoginFailureWindow, cfg.LoginLockDuration)
	revoked := redisstore.NewRevokedSessions(redisClient, cfg.AccessTokenTTL)

	accessTokens, err := accesstoken.NewIssuer(cfg.AccessTokenPrivateKey, cfg.AccessTokenTTL)
	if err != nil {
		return err
	}
	logger.Info("access token signing key loaded", "key_id", accessTokens.KeyID())

	dummyHash, err := newDummyHash(ctx, hasher)
	if err != nil {
		return err
	}

	// Use cases.
	sessionTokens := app.NewSessionTokens(refreshTokens, accessTokens, cfg.RefreshTokenTTL)

	handler := grpcapi.NewHandler(grpcapi.UseCases{
		RegisterUser: app.NewRegisterUser(users, hasher, utcNow),
		Login: app.NewLogin(app.LoginDeps{
			Users:      users,
			Hasher:     hasher,
			Sessions:   sessions,
			Attempts:   loginAttempts,
			Limiter:    limiter,
			Tokens:     sessionTokens,
			Tx:         db,
			Logger:     logger,
			Now:        utcNow,
			SessionTTL: cfg.SessionTTL,
			DummyHash:  dummyHash,
		}),
		RefreshSession: app.NewRefreshSession(app.RefreshSessionDeps{
			RefreshTokens: refreshTokens,
			Sessions:      sessions,
			Tokens:        sessionTokens,
			Revoked:       revoked,
			Tx:            db,
			Logger:        logger,
			Now:           utcNow,
		}),
		Logout: app.NewLogout(sessions, revoked, utcNow),
	}, logger)

	server := grpcserver.NewServer(logger, pool)
	identityv1.RegisterIdentityServiceServer(server, handler)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return err
	}

	return server.Run(ctx, lis)
}

func newRedisClient(ctx context.Context, url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("redis: parse url: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return client, nil
}

// newDummyHash makes the hash that Login verifies against when the
// identifier matches nobody. Its password is random and thrown away.
func newDummyHash(ctx context.Context, hasher *passwordhash.Argon2id) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	hash, err := hasher.Hash(ctx, rand.Text())
	if err != nil {
		return "", fmt.Errorf("create dummy password hash: %w", err)
	}
	return hash, nil
}

func utcNow() time.Time {
	return time.Now().UTC()
}
