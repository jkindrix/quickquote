# QuickQuote Makefile
# Comprehensive build, test, and deployment automation

.PHONY: all build run test clean dev help
.PHONY: deps fmt lint vet security
.PHONY: db-up db-down db-shell db-backup db-restore
.PHONY: migrate-up migrate-down migrate-status migrate-create migrate-force
.PHONY: docker-build docker-up docker-down docker-logs docker-shell docker-clean
.PHONY: prod-deploy prod-build prod-up prod-down prod-logs prod-restart prod-status
.PHONY: prod-migrate prod-backup prod-shell
.PHONY: test-unit test-integration test-coverage test-race test-short
.PHONY: setup check-env install-tools

# ==============================================================================
# Configuration
# ==============================================================================

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GORUN := $(GOCMD) run
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Application
BINARY_NAME := quickquote
MAIN_PATH := ./cmd/server
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Docker
DOCKER_COMPOSE := docker-compose
DOCKER_COMPOSE_DEV := $(DOCKER_COMPOSE) -f docker-compose.dev.yml
DOCKER_COMPOSE_PROD := $(DOCKER_COMPOSE) -f docker-compose.prod.yml
APP_CONTAINER := quickquote-app
DB_CONTAINER := quickquote-db

# Database
DATABASE_URL ?= postgres://quickquote:quickquote@localhost:5432/quickquote?sslmode=disable
MIGRATIONS_PATH := migrations

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

# ==============================================================================
# Main Targets
# ==============================================================================

## all: Build the application (default)
all: build

## build: Build the application binary
build:
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Build complete: ./$(BINARY_NAME)$(NC)"

## build-linux: Build for Linux (for Docker)
build-linux:
	@echo "$(GREEN)Building for Linux...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_NAME) $(MAIN_PATH)

## run: Run the application locally
run: build
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	./$(BINARY_NAME)

## dev: Run with hot reload (requires air)
dev:
	@command -v air >/dev/null 2>&1 || { echo "$(RED)air is not installed. Run: go install github.com/cosmtrek/air@latest$(NC)"; exit 1; }
	air

## clean: Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf tmp/
	@echo "$(GREEN)Clean complete$(NC)"

# ==============================================================================
# Dependencies & Code Quality
# ==============================================================================

## deps: Download and tidy dependencies
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) tidy
	$(GOMOD) verify

## fmt: Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) ./...
	@echo "$(GREEN)Format complete$(NC)"

## lint: Run linter (requires golangci-lint)
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "$(RED)golangci-lint is not installed$(NC)"; exit 1; }
	@echo "$(GREEN)Running linter...$(NC)"
	golangci-lint run --timeout 5m

## vet: Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GOVET) ./...

## security: Run security scanner (requires gosec)
security:
	@command -v gosec >/dev/null 2>&1 || { echo "$(YELLOW)Installing gosec...$(NC)"; go install github.com/securego/gosec/v2/cmd/gosec@latest; }
	@echo "$(GREEN)Running security scan...$(NC)"
	gosec -quiet ./...

## check: Run all code quality checks
check: fmt vet lint security
	@echo "$(GREEN)All checks passed$(NC)"

# ==============================================================================
# Testing
# ==============================================================================

## test: Run all tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v ./...

## test-short: Run tests in short mode
test-short:
	@echo "$(GREEN)Running short tests...$(NC)"
	$(GOTEST) -short -v ./...

## test-unit: Run unit tests only (exclude integration)
test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	$(GOTEST) -v -short ./...

## test-integration: Run integration tests
test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	$(GOTEST) -v -run Integration ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GOTEST) -v -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

## test-race: Run tests with race detector
test-race:
	@echo "$(GREEN)Running tests with race detector...$(NC)"
	$(GOTEST) -v -race ./...

# ==============================================================================
# Development Database (Local)
# ==============================================================================

## db-up: Start development database
db-up:
	@echo "$(GREEN)Starting development database...$(NC)"
	$(DOCKER_COMPOSE_DEV) up -d
	@echo "$(GREEN)Database started$(NC)"

## db-down: Stop development database
db-down:
	@echo "$(YELLOW)Stopping development database...$(NC)"
	$(DOCKER_COMPOSE_DEV) down

## db-shell: Open psql shell to development database
db-shell:
	@echo "$(GREEN)Connecting to database...$(NC)"
	docker exec -it $(DB_CONTAINER) psql -U quickquote -d quickquote

## db-logs: View database logs
db-logs:
	docker logs -f $(DB_CONTAINER)

# ==============================================================================
# Migrations
# ==============================================================================

## migrate-up: Run all pending migrations
migrate-up:
	@command -v migrate >/dev/null 2>&1 || { echo "$(RED)migrate is not installed. Run: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest$(NC)"; exit 1; }
	@echo "$(GREEN)Running migrations...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up
	@echo "$(GREEN)Migrations complete$(NC)"

## migrate-down: Rollback last migration
migrate-down:
	@echo "$(YELLOW)Rolling back last migration...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

## migrate-down-all: Rollback all migrations (DANGEROUS)
migrate-down-all:
	@echo "$(RED)WARNING: This will rollback ALL migrations!$(NC)"
	@read -p "Are you sure? [y/N] " confirm && [ "$$confirm" = "y" ]
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down -all

## migrate-status: Show migration status
migrate-status:
	@echo "$(GREEN)Migration status:$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version

## migrate-create: Create a new migration (usage: make migrate-create name=add_users_table)
migrate-create:
	@test -n "$(name)" || { echo "$(RED)Usage: make migrate-create name=migration_name$(NC)"; exit 1; }
	@echo "$(GREEN)Creating migration: $(name)$(NC)"
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)
	@echo "$(GREEN)Created new migration files$(NC)"

## migrate-force: Force set migration version (usage: make migrate-force version=5)
migrate-force:
	@test -n "$(version)" || { echo "$(RED)Usage: make migrate-force version=N$(NC)"; exit 1; }
	@echo "$(YELLOW)Forcing migration version to $(version)...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $(version)

# ==============================================================================
# Docker Development
# ==============================================================================

## docker-build: Build Docker image
docker-build:
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -t quickquote:latest .
	@echo "$(GREEN)Docker image built$(NC)"

## docker-up: Start all services with Docker Compose
docker-up:
	@echo "$(GREEN)Starting services...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Services started$(NC)"

## docker-down: Stop all services
docker-down:
	@echo "$(YELLOW)Stopping services...$(NC)"
	$(DOCKER_COMPOSE) down

## docker-logs: View application logs
docker-logs:
	docker logs -f $(APP_CONTAINER)

## docker-shell: Open shell in app container
docker-shell:
	docker exec -it $(APP_CONTAINER) sh

## docker-clean: Remove all containers, images, and volumes
docker-clean:
	@echo "$(RED)WARNING: This will remove all quickquote containers and volumes!$(NC)"
	@read -p "Are you sure? [y/N] " confirm && [ "$$confirm" = "y" ]
	$(DOCKER_COMPOSE) down -v --rmi local
	$(DOCKER_COMPOSE_PROD) down -v --rmi local 2>/dev/null || true

# ==============================================================================
# Production Deployment
# ==============================================================================

## prod-deploy: Full production deployment (build + migrate + restart)
prod-deploy: prod-build prod-migrate prod-restart prod-status
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(GREEN)Production deployment complete!$(NC)"
	@echo "$(GREEN)========================================$(NC)"

## prod-build: Build production Docker image
prod-build:
	@echo "$(GREEN)Building production image...$(NC)"
	$(DOCKER_COMPOSE_PROD) build --no-cache
	@echo "$(GREEN)Production image built$(NC)"

## prod-up: Start production services
prod-up:
	@echo "$(GREEN)Starting production services...$(NC)"
	$(DOCKER_COMPOSE_PROD) up -d
	@echo "$(GREEN)Production services started$(NC)"

## prod-down: Stop production services
prod-down:
	@echo "$(YELLOW)Stopping production services...$(NC)"
	$(DOCKER_COMPOSE_PROD) down

## prod-restart: Restart production services (rebuild + up)
prod-restart:
	@echo "$(GREEN)Restarting production services...$(NC)"
	$(DOCKER_COMPOSE_PROD) up -d --build
	@echo "$(GREEN)Production services restarted$(NC)"

## prod-logs: View production logs
prod-logs:
	$(DOCKER_COMPOSE_PROD) logs -f app

## prod-status: Show production status
prod-status:
	@echo "$(GREEN)Production Status:$(NC)"
	@echo ""
	@docker ps --filter "name=quickquote" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "$(GREEN)Health Check:$(NC)"
	@docker exec $(APP_CONTAINER) wget -qO- http://127.0.0.1:8080/health 2>/dev/null | jq . || echo "$(YELLOW)Service health endpoint unavailable from host$(NC)"

## prod-shell: Open shell in production app container
prod-shell:
	docker exec -it $(APP_CONTAINER) sh

## prod-db-shell: Open psql shell to production database
prod-db-shell:
	docker exec -it $(DB_CONTAINER) psql -U quickquote -d quickquote

## prod-migrate: Run migrations on production
prod-migrate:
	@echo "$(GREEN)Running production migrations...$(NC)"
	@docker exec $(DB_CONTAINER) psql -U quickquote -d quickquote -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;" 2>/dev/null || echo "No migrations yet"
	$(DOCKER_COMPOSE_PROD) exec -T app sh -c 'cd /app && migrate -path migrations -database "postgres://quickquote:$$DATABASE_PASSWORD@quickquote-db:5432/quickquote?sslmode=disable" up' 2>/dev/null || \
		echo "$(YELLOW)Note: Run migrations manually if migrate tool not in container$(NC)"

## prod-backup: Backup production database
prod-backup:
	@echo "$(GREEN)Backing up production database...$(NC)"
	@mkdir -p backups
	docker exec $(DB_CONTAINER) pg_dump -U quickquote -d quickquote > backups/quickquote_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)Backup saved to backups/$(NC)"
	@ls -la backups/*.sql | tail -5

## prod-restore: Restore production database (usage: make prod-restore file=backups/quickquote_20240101.sql)
prod-restore:
	@test -n "$(file)" || { echo "$(RED)Usage: make prod-restore file=backups/quickquote_YYYYMMDD.sql$(NC)"; exit 1; }
	@test -f "$(file)" || { echo "$(RED)File not found: $(file)$(NC)"; exit 1; }
	@echo "$(RED)WARNING: This will overwrite the production database!$(NC)"
	@read -p "Are you sure? [y/N] " confirm && [ "$$confirm" = "y" ]
	docker exec -i $(DB_CONTAINER) psql -U quickquote -d quickquote < $(file)
	@echo "$(GREEN)Database restored from $(file)$(NC)"

# ==============================================================================
# Setup & Installation
# ==============================================================================

## setup: Initial project setup
setup: deps install-tools
	@echo "$(GREEN)Setup complete!$(NC)"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Copy .env.example to .env and configure"
	@echo "  2. Run 'make db-up' to start the database"
	@echo "  3. Run 'make migrate-up' to run migrations"
	@echo "  4. Run 'make dev' to start development server"

## install-tools: Install development tools
install-tools:
	@echo "$(GREEN)Installing development tools...$(NC)"
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "$(GREEN)Tools installed$(NC)"

## check-env: Verify environment configuration
check-env:
	@echo "$(GREEN)Checking environment...$(NC)"
	@echo ""
	@echo "Go version: $$(go version)"
	@echo "Docker version: $$(docker --version 2>/dev/null || echo 'Not installed')"
	@echo "Docker Compose version: $$(docker-compose --version 2>/dev/null || echo 'Not installed')"
	@echo ""
	@echo "Required tools:"
	@command -v air >/dev/null 2>&1 && echo "  ✓ air" || echo "  ✗ air (run: make install-tools)"
	@command -v golangci-lint >/dev/null 2>&1 && echo "  ✓ golangci-lint" || echo "  ✗ golangci-lint"
	@command -v migrate >/dev/null 2>&1 && echo "  ✓ migrate" || echo "  ✗ migrate"
	@command -v gosec >/dev/null 2>&1 && echo "  ✓ gosec" || echo "  ✗ gosec"
	@echo ""
	@test -f .env && echo "$(GREEN)✓ .env file exists$(NC)" || echo "$(YELLOW)✗ .env file missing (copy from .env.example)$(NC)"

# ==============================================================================
# Utilities
# ==============================================================================

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"

## loc: Count lines of code
loc:
	@echo "$(GREEN)Lines of code:$(NC)"
	@find . -name '*.go' -not -path './vendor/*' | xargs wc -l | tail -1

## todo: Find TODO comments in code
todo:
	@echo "$(GREEN)TODO items in code:$(NC)"
	@grep -rn "TODO" --include="*.go" . || echo "No TODOs found"

# ==============================================================================
# Help
# ==============================================================================

## help: Show this help message
help:
	@echo ""
	@echo "$(GREEN)QuickQuote - Voice AI Quote Generator$(NC)"
	@echo "$(GREEN)======================================$(NC)"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "$(YELLOW)Main Targets:$(NC)"
	@grep -E '^## ' $(MAKEFILE_LIST) | grep -E '(build|run|dev|test|clean):' | sed 's/## /  /' | sed 's/:/:	/'
	@echo ""
	@echo "$(YELLOW)Code Quality:$(NC)"
	@grep -E '^## ' $(MAKEFILE_LIST) | grep -E '(deps|fmt|lint|vet|security|check):' | sed 's/## /  /' | sed 's/:/:	/'
	@echo ""
	@echo "$(YELLOW)Testing:$(NC)"
	@grep -E '^## ' $(MAKEFILE_LIST) | grep -E 'test-' | sed 's/## /  /' | sed 's/:/:	/'
	@echo ""
	@echo "$(YELLOW)Database:$(NC)"
	@grep -E '^## ' $(MAKEFILE_LIST) | grep -E '(db-|migrate-)' | sed 's/## /  /' | sed 's/:/:	/'
	@echo ""
	@echo "$(YELLOW)Docker Development:$(NC)"
	@grep -E '^## ' $(MAKEFILE_LIST) | grep -E 'docker-' | sed 's/## /  /' | sed 's/:/:	/'
	@echo ""
	@echo "$(YELLOW)Production:$(NC)"
	@grep -E '^## ' $(MAKEFILE_LIST) | grep -E 'prod-' | sed 's/## /  /' | sed 's/:/:	/'
	@echo ""
	@echo "$(YELLOW)Setup & Utilities:$(NC)"
	@grep -E '^## ' $(MAKEFILE_LIST) | grep -E '(setup|install-tools|check-env|version|loc|todo):' | sed 's/## /  /' | sed 's/:/:	/'
	@echo ""
	@echo "$(GREEN)Quick Start:$(NC)"
	@echo "  make setup          # Initial setup"
	@echo "  make dev            # Start development"
	@echo "  make prod-deploy    # Deploy to production"
	@echo ""
