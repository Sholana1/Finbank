GOBIN := $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

MODULES := $(shell go list -m -f '{{.Dir}}/...')
COMPOSE := docker compose -f deploy/local/docker-compose.yml

LEDGER_DB_URL ?= postgres://ledger_app:ledger_app_local@localhost:5432/ledger?sslmode=disable
LEDGER_MIGRATION_DB_URL ?= postgres://ledger_owner:ledger_owner_local@localhost:5432/ledger?sslmode=disable
IDENTITY_DB_URL ?= postgres://identity_app:identity_app_local@localhost:5432/identity?sslmode=disable
IDENTITY_MIGRATION_DB_URL ?= postgres://identity_owner:identity_owner_local@localhost:5432/identity?sslmode=disable

.PHONY: test test-integration vet proto infra-up infra-down infra-reset \
        migrate migrate-ledger migrate-identity run-ledger run-identity

test:
	go test -race -count=1 $(MODULES)

test-integration:
	IDENTITY_TEST_DATABASE_URL='$(IDENTITY_DB_URL)' go test -race -count=1 $(MODULES)

vet:
	go vet $(MODULES)

proto:
	PATH="$(GOBIN):$$PATH" buf lint
	PATH="$(GOBIN):$$PATH" buf generate

infra-up:
	$(COMPOSE) up -d --wait

infra-down:
	$(COMPOSE) down

infra-reset:
	$(COMPOSE) down -v

migrate: migrate-ledger migrate-identity

migrate-ledger:
	MIGRATION_DATABASE_URL='$(LEDGER_MIGRATION_DB_URL)' go run ./services/ledger/cmd/migrate

migrate-identity:
	MIGRATION_DATABASE_URL='$(IDENTITY_MIGRATION_DB_URL)' go run ./services/identity/cmd/migrate

run-ledger:
	DATABASE_URL='$(LEDGER_DB_URL)' go run ./services/ledger/cmd/server

run-identity:
	DATABASE_URL='$(IDENTITY_DB_URL)' go run ./services/identity/cmd/server
