package config

import (
	"errors"
	"os"
)

type Config struct {
	ServiceName string
	Env         string
	GRPCAddr    string
	DatabaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		ServiceName: "identity",
		Env:         getenv("APP_ENV", "local"),
		GRPCAddr:    getenv("GRPC_ADDR", ":50052"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("config: DATABASE_URL is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func MigrationDatabaseURL() (string, error) {
	url := os.Getenv("MIGRATION_DATABSE_URL")
	if url == "" {
		return "", errors.New("config: MIGRATION_DATABASE_URL is required")
	}
	return url, nil
}
