# Makefile for voidline CLI and server

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOOSE=go run github.com/pressly/goose/v3@latest
SWAG=go run github.com/swaggo/swag/cmd/swag@v1.16.6
AIR_VERSION=1.64.5
GOLANGCI_LINT_VERSION=1.64.8
GOFUMPT_VERSION=0.9.2
GOIMPORTS_VERSION=0.21.1

# Directories
BINDIR=bin
SRCDIR=.

# Binary name
BINARY_NAME=voidline

# Database
DB_PATH ?= $(HOME)/.config/hominem/db.sqlite
MIGRATIONS_DIR = internal/infrastructure/persistence/sqlite/migrations

# Default target
all: build

# Build the project
build:
	$(GOBUILD) -o $(BINDIR)/$(BINARY_NAME) ./cmd/cli

# Build server only
build-server:
	$(GOBUILD) -o $(BINDIR)/server ./cmd/server

# Clean the project
clean:
	$(GOCLEAN)
	rm -f $(BINDIR)/$(BINARY_NAME)
	rm -f $(BINDIR)/server

# Run tests
test:
	$(GOTEST) -v ./...

# Install dev tools
tools:
	$(GOCMD) install github.com/swaggo/swag/cmd/swag@v1.16.6
	$(GOCMD) install github.com/air-verse/air@v$(AIR_VERSION)
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)
	$(GOCMD) install mvdan.cc/gofumpt@v$(GOFUMPT_VERSION)
	$(GOCMD) install golang.org/x/tools/cmd/goimports@v$(GOIMPORTS_VERSION)

# Tidy up the Go module
tidy:
	$(GOMOD) tidy

# Generate Swagger docs
swagger:
	$(SWAG) init -g cmd/cli/commands/server/main.go -o docs

# Get dependencies
deps:
	$(GOGET) -u ./...

# Format Go code
fmt:
	@gofumpt -w $$(go list -f '{{.Dir}}' ./...)
	@goimports -w $$(go list -f '{{.Dir}}' ./...)

# Lint Go code
lint:
	@golangci-lint run ./...

# Run the CLI
run: build
	./$(BINDIR)/$(BINARY_NAME)

# Run the server
run-server: build-server
	./$(BINDIR)/server

# Development server with live reload + swagger
dev:
	@air -c .air.toml

# Install goose
install-goose:
	$(GOGET) github.com/pressly/goose/v3/cmd/goose

# Migration commands
migrate-up:
	$(GOOSE) -dir $(MIGRATIONS_DIR) sqlite3 $(DB_PATH) up

migrate-down:
	$(GOOSE) -dir $(MIGRATIONS_DIR) sqlite3 $(DB_PATH) down

migrate-create:
	$(GOOSE) -dir $(MIGRATIONS_DIR) sqlite3 $(DB_PATH) create $(NAME) sql

migrate-status:
	$(GOOSE) -dir $(MIGRATIONS_DIR) sqlite3 $(DB_PATH) status

migrate-reset:
	$(GOOSE) -dir $(MIGRATIONS_DIR) sqlite3 $(DB_PATH) reset

# Help message
help:
	@echo "Makefile for voidline CLI and server"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  all              Build the project"
	@echo "  build            Build the CLI"
	@echo "  build-server     Build the server"
	@echo "  clean            Clean the project"
	@echo "  test             Run tests"
	@echo "  tools            Install dev tools"
	@echo "  tidy             Tidy up the Go module"
	@echo "  deps             Get dependencies"
	@echo "  fmt              Format Go code"
	@echo "  lint             Lint Go code"
	@echo "  swagger          Generate Swagger docs"
	@echo "  run              Run the CLI"
	@echo "  run-server       Run the server"
	@echo "  dev              Run server with live reload"
	@echo "  install-goose    Install goose"
	@echo "  migrate-up       Run migrations up"
	@echo "  migrate-down     Run migrations down"
	@echo "  migrate-create   Create new migration (NAME=foo)"
	@echo "  migrate-status   Show migration status"
	@echo "  migrate-reset    Reset all migrations"
	@echo ""
	@echo "Examples:"
	@echo "  make migrate-up DB_PATH=~/path/to/db.sqlite3"
	@echo "  make migrate-create NAME=add_users_table"
	@echo "  make swagger"
