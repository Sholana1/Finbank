package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName string
	Env         string
	GRPCAddr    string
	DatabaseURL string
	RedisURL    string

	// AccessTokenPrivateKey is a secret. Never log it.
	AccessTokenPrivateKey string
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
	SessionTTL            time.Duration

	LoginMaxFailures   int
	LoginFailureWindow time.Duration
	LoginLockDuration  time.Duration

	PasswordHashConcurrency int
}

// Load reads every setting and reports all problems together, so one
// run tells you everything that is missing or malformed.
func Load() (Config, error) {
	var errs []error

	cfg := Config{
		ServiceName: "identity",
		Env:         getenv("APP_ENV", "local"),
		GRPCAddr:    getenv("GRPC_ADDR", ":50052"),
		DatabaseURL: required("DATABASE_URL", &errs),
		RedisURL:    required("REDIS_URL", &errs),

		AccessTokenPrivateKey: required("ACCESS_TOKEN_PRIVATE_KEY", &errs),
		AccessTokenTTL:        duration("ACCESS_TOKEN_TTL", 15*time.Minute, &errs),
		RefreshTokenTTL:       duration("REFRESH_TOKEN_TTL", 7*24*time.Hour, &errs),
		SessionTTL:            duration("SESSION_TTL", 30*24*time.Hour, &errs),

		LoginMaxFailures:   positiveInt("LOGIN_MAX_FAILURES", 5, &errs),
		LoginFailureWindow: duration("LOGIN_FAILURE_WINDOW", 15*time.Minute, &errs),
		LoginLockDuration:  duration("LOGIN_LOCK_DURATION", 15*time.Minute, &errs),

		PasswordHashConcurrency: positiveInt("PASSWORD_HASH_CONCURRENCY", 4, &errs),
	}

	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// MigrationDatabaseURL returns the connection string for the schema-owner
// role. Only cmd/migrate uses it; the running service never sees it.
func MigrationDatabaseURL() (string, error) {
	url := os.Getenv("MIGRATION_DATABASE_URL")
	if url == "" {
		return "", errors.New("config: MIGRATION_DATABASE_URL is required")
	}
	return url, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func required(key string, errs *[]error) string {
	v := os.Getenv(key)
	if v == "" {
		*errs = append(*errs, fmt.Errorf("config: %s is required", key))
	}
	return v
}

func duration(key string, fallback time.Duration, errs *[]error) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		*errs = append(*errs, fmt.Errorf("config: %s must be a positive duration such as 15m", key))
		return fallback
	}
	return d
}

func positiveInt(key string, fallback int, errs *[]error) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		*errs = append(*errs, fmt.Errorf("config: %s must be a positive whole number", key))
		return fallback
	}
	return n
}
