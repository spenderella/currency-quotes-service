include .env

MIGRATIONS_DIR = internal/db/postgres
DSN = host=$(POSTGRES_HOST) port=$(POSTGRES_PORT) user=$(POSTGRES_USER) password=$(POSTGRES_PASSWORD) dbname=$(POSTGRES_NAME) sslmode=$(POSTGRES_SSLMODE)

.PHONY: migrate-up migrate-down migrate-status

migrate-up:
	goose postgres "$(DSN)" -dir $(MIGRATIONS_DIR) up

migrate-down:
	goose postgres "$(DSN)" -dir $(MIGRATIONS_DIR) down

migrate-status:
	goose postgres "$(DSN)" -dir $(MIGRATIONS_DIR) status
