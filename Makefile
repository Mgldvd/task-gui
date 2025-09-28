# Makefile for Task Runner GUI

# Variables
BINARY_NAME=taskg
BUILD_DIR=.
CMD_DIR=./cmd/taskg
VERSION=1.0.0

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test ./internal/...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)

# Install the binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Uninstall the binary
.PHONY: uninstall
uninstall:
	@echo "Removing $(BINARY_NAME) from /usr/local/bin..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

# Run the application with default settings
.PHONY: run
run: build
	./$(BINARY_NAME)

# Run with example commands
.PHONY: run-example
run-example: build
	./$(BINARY_NAME) --commands=example-commands.json

# Run with light theme
.PHONY: run-light
run-light: build
	./$(BINARY_NAME) --theme=light

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        Build the application"
	@echo "  deps         Install dependencies"
	@echo "  test         Run tests"
	@echo "  clean        Clean build artifacts"
	@echo "  install      Install binary to /usr/local/bin"
	@echo "  uninstall    Remove binary from /usr/local/bin"
	@echo "  run          Run with default embedded commands"
	@echo "  run-example  Run with example-commands.json"
	@echo "  run-light    Run with light theme"
	@echo "  fmt          Format code"
	@echo "  lint         Run linter"
	@echo "  help         Show this help"