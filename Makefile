# Makefile for gogogo CLI and server

# Variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOOSE=go run github.com/pressly/goose/v3@latest

# Directories
BINDIR=bin
SRCDIR=.

# Binary name
BINARY_NAME=gogogo

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
	$(GOBUILD) -o $(BINDIR)/server ./cmd/cli

# Clean the project
clean:
	$(GOCLEAN)
	rm -f $(BINDIR)/$(BINARY_NAME)

# Run tests
test:
	$(GOTEST) -v ./...

# Tidy up the Go module
tidy:
	$(GOMOD) tidy

# Get dependencies
deps:
	$(GOGET) -u ./...

# Run the CLI
run: build
	./$(BINDIR)/$(BINARY_NAME)

# Run the server
run-server: build-server
	./$(BINDIR)/server

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
	@echo "Makefile for gogogo CLI and server"
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
	@echo "  tidy             Tidy up the Go module"
	@echo "  deps             Get dependencies"
	@echo "  run              Run the CLI"
	@echo "  run-server       Run the server"
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
