GOOSE := goose
MIGRATIONS_DIR := internal/db/migrations
APP_PATH := ./cmd/api/

.PHONY: help
help:
	@echo "Usage:"
	@echo "  make run 					Run go app"
	@echo "  make migrate-up            Run all up migrations"
	@echo "  make migrate-down         Rollback the last migration"
	@echo "  make migrate-status       Show migration status"
	@echo "  make migrate-create name=your_migration_name"
	@echo "                            Create a new migration"

.PHONY: run
run:
	go run $(APP_PATH)

.PHONY: migrate-up
migrate-up:
	$(GOOSE) -dir $(MIGRATIONS_DIR) up

.PHONY: migrate-down
	$(GOOSE) -dir $(MIGRATIONS_DIR) down

.PHONY: migrate-status
migrate-status:
	$(GOOSE) -dir $(MIGRATIONS_DIR) status

.PHONY: migrate-create
migrate-create:
ifndef name
	$(error name is required. Usage: make migrate-create name=add_users_table)
endif
	$(GOOSE) -dir $(MIGRATIONS_DIR) create $(name) sql

