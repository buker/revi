# revi - AI Code Review & Commit
# Makefile for building and installing

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build settings
BINARY_NAME = revi
BUILD_DIR = bin
GO = go
GOFLAGS = -trimpath
LDFLAGS = -s -w \
	-X 'github.com/buker/revi/internal/cli.Version=$(VERSION)'

# Installation paths
PREFIX ?= /usr/local
INSTALL_DIR = $(PREFIX)/bin

.PHONY: all build install uninstall clean test lint help

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/revi

# Install to system
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed successfully!"

# Uninstall from system
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_DIR)..."
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalled successfully!"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean!"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Upgrade Go dependencies
deps-upgrade:
	@echo "Upgrading Go dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✅ Go dependencies upgraded"

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, install it first" && exit 1)
	golangci-lint run ./...

# Development build (no optimization)
dev:
	@echo "Building development version..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/revi

# Show help
help:
	@echo "revi - AI Code Review & Commit"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build the binary"
	@echo "  make build        Build the binary"
	@echo "  make install      Install to $(INSTALL_DIR)"
	@echo "  make uninstall    Remove from $(INSTALL_DIR)"
	@echo "  make clean        Remove build artifacts"
	@echo "  make test         Run tests"
	@echo "  make lint         Run linter"
	@echo "  make dev          Build development version"
	@echo "  make help         Show this help"
	@echo "  make deps-upgrade Upgrade Go dependencies"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX            Installation prefix (default: /usr/local)"
	@echo "  VERSION           Version string (default: git describe)"
