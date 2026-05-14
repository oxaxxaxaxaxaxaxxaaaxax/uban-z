SHELL := /bin/bash

MIGRATIONS_DIR := migrations/booking
SQLC_DIR       := internal/adapter/booking/postgres
DATABASE_URL   ?= postgres://booking:booking@localhost:5432/booking?sslmode=disable

BOOKING_PKGS := ./cmd/booking/... ./internal/... ./test/...

.PHONY: help build test test-integration test-all vet migrate-up migrate-down migrate-status sqlc-generate compose-up compose-down

help:
	@echo "Targets:"
	@echo "  build              Build the booking binary into ./bin/booking"
	@echo "  test               Run unit tests (-race)"
	@echo "  test-integration   Run integration tests (Docker required)"
	@echo "  test-all           Run unit + integration tests"
	@echo "  vet                Run go vet across booking packages"
	@echo "  migrate-up         Apply pending goose migrations against \$$DATABASE_URL"
	@echo "  migrate-down       Roll the last goose migration back"
	@echo "  migrate-status     Show migration state"
	@echo "  sqlc-generate      Regenerate sqlc code from query.sql"
	@echo "  compose-up         Bring up the booking-only docker-compose slice"
	@echo "  compose-down       Tear down the slice (and named volumes)"

build:
	@mkdir -p bin
	@go build -trimpath -o bin/booking ./cmd/booking
	@echo "built bin/booking"

test:
	@go test -race -count=1 $(BOOKING_PKGS)

test-integration:
	@go test -tags=integration -race -count=1 -timeout=10m -p 1 $(BOOKING_PKGS)

test-all: test test-integration

vet:
	@go vet $(BOOKING_PKGS)

migrate-up:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status:
	@goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

sqlc-generate:
	@cd $(SQLC_DIR) && sqlc generate

compose-up:
	@docker compose -f cmd/booking/compose.yaml up --build -d

compose-down:
	@docker compose -f cmd/booking/compose.yaml down -v
