MODULES := $(shell go list -m -f '{{.Dir}}/...')
COMPOSE := docker compose -f deploy/local/docker-compose.yml

LEDGER_DB_URL ?= postgres://ledger_app:ledger_app_local@localhost:5432/ledger?sslmode=disable
LEDGER_MIGRATION_DB_URL ?= postgres://ledger_owner:ledger_owner_local@localhost:5432/ledger?sslmode=disable

.PHONY: test vet infra-up infra-down infra-reset migrate-ledger run-ledger psql-ledger

test:
	go test -race -count=1 $(MODULES)

vet:
	go vet $(MODULES)

infra-up:
	$(COMPOSE) up -d --wait

infra-down:
	$(COMPOSE) down

infra-reset:
	$(COMPOSE) down -v

migrate-ledger:
	MIGRATION_DATABASE_URL='$(LEDGER_MIGRATION_DB_URL)' go run ./services/ledger/cmd/migrate

run-ledger:
	DATABASE_URL='$(LEDGER_DB_URL)' go run ./services/ledger/cmd/server

psql-ledger:
	$(COMPOSE) exec postgres psql -U ledger_app -d ledger
