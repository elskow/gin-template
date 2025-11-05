ifneq (,$(wildcard ./.env))
	include .env
	export $(shell sed 's/=.*//' .env)
endif

CONTAINER_NAME=${APP_NAME}-app
POSTGRES_CONTAINER_NAME=${APP_NAME}-db

# Development Commands
.PHONY: run build

run:
	@go run cmd/main.go

build:
	@go build -o bin/server cmd/main.go

# Testing Commands
.PHONY: test test-verbose test-race test-coverage

test:
	@go test ./...

test-verbose:
	@go test -v ./...

test-race:
	@go test -race ./...

test-coverage:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

# Benchmark Commands
.PHONY: bench bench-mem

bench:
	@go test -bench=. -run=^$$ ./...

bench-mem:
	@go test -bench=. -benchmem -run=^$$ ./...

# Module Generation
.PHONY: module rename

module:
	@if [ -z "$(name)" ]; then \
		echo "Error: module name required"; \
		echo "Usage: make module name=<module_name>"; \
		exit 1; \
	fi
	@./script/create_module.sh $(name)

rename:
	@if [ -z "$(name)" ]; then \
		echo "Error: new Go module path required"; \
		echo "Usage: make rename name=<new-module-path>"; \
		echo "Example: make rename name=github.com/yourorg/yourproject"; \
		exit 1; \
	fi
	@./script/rename_project.sh $(name)

# Database Commands
.PHONY: migrate seed

migrate:
	@go run cmd/main.go --migrate

seed:
	@go run cmd/main.go --seed

# Docker Commands
.PHONY: dev-up dev-down

dev-up:
	@docker-compose -f docker-compose.dev.yml up -d --build

dev-down:
	@docker-compose -f docker-compose.dev.yml down

# Staging Commands
.PHONY: staging-up staging-down

staging-up:
	@docker-compose -f docker-compose.staging.yml up -d --build

staging-down:
	@docker-compose -f docker-compose.staging.yml down

# Utility Commands
.PHONY: clean

clean:
	@rm -rf bin/
	@rm -f coverage.out
	@rm -f main

help:
	@echo "Available commands:"
	@echo ""
	@echo "Development:"
	@echo "  make run              - Run server"
	@echo "  make build            - Build binary"
	@echo ""
	@echo "Testing:"
	@echo "  make test             - Run tests"
	@echo "  make test-verbose     - Run tests with verbose output"
	@echo "  make test-race        - Run tests with race detector"
	@echo "  make test-coverage    - Run tests with coverage"
	@echo ""
	@echo "Benchmarking:"
	@echo "  make bench            - Run benchmarks"
	@echo "  make bench-mem        - Run benchmarks with memory stats"
	@echo ""
	@echo "Module:"
	@echo "  make module name=<name> - Create new module"
	@echo "  make rename name=<path> - Rename Go module path"
	@echo ""
	@echo "Database:"
	@echo "  make migrate          - Run migrations"
	@echo "  make seed             - Run seeders"
	@echo ""
	@echo "Docker:"
	@echo "  make dev-up           - Start dev environment"
	@echo "  make dev-down         - Stop dev environment"
	@echo "  make staging-up       - Start staging environment"
	@echo "  make staging-down     - Stop staging environment"
	@echo ""
	@echo "Utility:"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make help             - Show this help"
