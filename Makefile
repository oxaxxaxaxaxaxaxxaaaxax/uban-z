SHELL := /bin/bash

MIGRATIONS_DIR := migrations/booking
AUTH_MIGRATIONS_DIR := migrations/auth
SQLC_DIR       := internal/adapter/booking/postgres
DATABASE_URL   ?= postgres://booking:booking@localhost:5432/booking?sslmode=disable
AUTH_DATABASE_URL ?= $(DATABASE_URL)

.PHONY: migrate-up migrate-down migrate-status auth-migrate-up auth-migrate-down auth-migrate-status sqlc-generate

migrate-up:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

auth-migrate-up:
	@goose -dir $(AUTH_MIGRATIONS_DIR) postgres "$(AUTH_DATABASE_URL)" up

auth-migrate-down:
	@goose -dir $(AUTH_MIGRATIONS_DIR) postgres "$(AUTH_DATABASE_URL)" down

auth-migrate-status:
	@goose -dir $(AUTH_MIGRATIONS_DIR) postgres "$(AUTH_DATABASE_URL)" status

sqlc-generate:
	@cd $(SQLC_DIR) && sqlc generate
