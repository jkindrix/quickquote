.PHONY: all build run test clean dev db-up db-down migrate lint help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=quickquote
MAIN_PATH=./cmd/server

# Default target
all: build

## build: Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

## run: Run the application
run:
	$(GORUN) $(MAIN_PATH)

## dev: Run with hot reload (requires air)
dev:
	air

## test: Run tests
test:
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## clean: Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## db-up: Start development database
db-up:
	docker-compose -f docker-compose.dev.yml up -d

## db-down: Stop development database
db-down:
	docker-compose -f docker-compose.dev.yml down

## migrate-up: Run database migrations
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

## migrate-down: Rollback database migrations
migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

## migrate-create: Create a new migration (usage: make migrate-create name=migration_name)
migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

## create-admin: Create admin user (usage: make create-admin email=admin@example.com password=secret)
create-admin:
	go run scripts/create_admin.go -email $(email) -password $(password)

## docker-build: Build Docker image
docker-build:
	docker build -t quickquote:latest .

## docker-up: Start all services with Docker Compose
docker-up:
	docker-compose up -d

## docker-down: Stop all services
docker-down:
	docker-compose down

## docker-logs: View logs
docker-logs:
	docker-compose logs -f app

## lint: Run linter (requires golangci-lint)
lint:
	golangci-lint run

## fmt: Format code
fmt:
	$(GOCMD) fmt ./...

## help: Show this help message
help:
	@echo "QuickQuote - Voice AI Quote Generator"
	@echo ""
	@echo "Usage:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
