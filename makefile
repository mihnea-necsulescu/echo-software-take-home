.PHONY: help setup run test test-verbose db-up db-down

help:
	@echo "FireGo Wallet Service - Available commands:"
	@echo ""
	@echo "  setup        - Install dependencies and prepare environment"
	@echo "  run          - Start the application"
	@echo "  test         - Run all tests"
	@echo "  test-verbose - Run tests with verbose output"
	@echo "  db-up        - Start PostgreSQL database container"
	@echo "  db-down      - Stop PostgreSQL database container"

setup:
	@echo "Setting up development environment..."
	go version
	go mod download
	go mod verify
	@echo "Dependencies installed"

db-up:
	@echo "Starting PostgreSQL database..."
	docker-compose up -d
	@echo "Database started"
	@echo "Waiting for database to be ready..."
	@sleep 3

db-down:
	@echo "Stopping PostgreSQL database..."
	docker-compose down

run:
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found"; \
		exit 1; \
	fi
	@echo "Starting FireGo Wallet Service..."
	@export $$(grep -v '^#' .env | xargs) && go run cmd/main.go

test:
	@echo "Running all tests..."
	go test ./internal/fireblocks ./internal/handler

test-verbose:
	@echo "Running tests (verbose)..."
	go test -v ./internal/fireblocks ./internal/handler
