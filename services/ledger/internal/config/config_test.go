package config

import "testing"

func TestLoad(t *testing.T) {
	t.Run("fails without DATABASE_URL", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")

		if _, err := Load(); err == nil {
			t.Fatal("expected an error, got nil")
		}
	})

	t.Run("applied default", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://test")
		t.Setenv("APP_ENV", "")
		t.Setenv("GRPC_ADDR", "")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Env != "local" || cfg.GRPCAddr != ":50051" {
			t.Fatalf("defaults not applied")
		}
	})

	t.Run("environment override defaults", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://test")
		t.Setenv("APP_ENV", "staging")
		t.Setenv("GRPC_ADDR", ":50052")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Env != "staging" || cfg.GRPCAddr != ":50052" {
			t.Fatalf("defaults not applied")
		}
	})
}
