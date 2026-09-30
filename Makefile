SHELL := /bin/bash
VERSION ?= dev
GO_BUILD := docker compose -f compose.build.yaml run --rm --no-deps go-builder go build

MIGRATIONS_DIR := migrations/booking
AUTH_MIGRATIONS_DIR := migrations/auth
SQLC_DIR       := internal/adapter/booking/postgres
DATABASE_URL   ?= postgres://booking:booking@localhost:5432/booking?sslmode=disable
AUTH_DATABASE_URL ?= $(DATABASE_URL)

BOOKING_PKGS := ./cmd/booking/... ./internal/... ./test/...

.PHONY: help build-auth build-booking build-gateway clean test test-integration test-all vet migrate-up migrate-down migrate-status sqlc-generate compose-up compose-down

help:
	@echo "Targets:"
	@echo "  build-auth         Build build/auth/VERSION/auth (VERSION defaults to dev)"
	@echo "  build-booking      Build build/booking/VERSION/booking"
	@echo "  build-gateway      Build build/gateway/VERSION/gateway"
	@echo "  clean              Remove all locally built binary versions"
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

build-auth:
	@mkdir -p build/auth/$(VERSION)
	$(GO_BUILD) -o build/auth/$(VERSION)/auth ./cmd/auth_service

build-booking:
	@mkdir -p build/booking/$(VERSION)
	$(GO_BUILD) -o build/booking/$(VERSION)/booking ./cmd/booking

build-gateway:
	@mkdir -p build/gateway/$(VERSION)
	$(GO_BUILD) -o build/gateway/$(VERSION)/gateway ./cmd/app

clean:
	rm -rf build

test:
	@go test -race -count=1 $(BOOKING_PKGS)

test-integration:
	@go test -tags=integration -race -count=1 -timeout=10m -p 1 $(BOOKING_PKGS)

test-all: test test-integration

vet:
	@go vet $(BOOKING_PKGS)

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

compose-up:
	@docker compose --env-file cmd/booking/.env -f cmd/booking/compose.yaml up --no-build -d

compose-down:
	@docker compose -f cmd/booking/compose.yaml down -v
