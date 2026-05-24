SHELL := /bin/bash

MIGRATIONS_DIR := migrations/booking
SQLC_DIR       := internal/adapter/booking/postgres
DATABASE_URL   ?= postgres://booking:booking@localhost:5432/booking?sslmode=disable

.PHONY: migrate-up migrate-down migrate-status sqlc-generate

migrate-up:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

sqlc-generate:
	@cd $(SQLC_DIR) && sqlc generate
