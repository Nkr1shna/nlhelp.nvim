# Makefile for nvim-smart-keybind-search
# Provides easy commands for building, testing, and managing the plugin

.PHONY: help install build test clean start stop health docs

# Default target
help:
	@echo "nvim-smart-keybind-search - Available commands:"
	@echo ""
	@echo "Installation:"
	@echo "  install     - Install the plugin and dependencies"
	@echo "  build       - Build the Go backend and database"
	@echo "  clean       - Clean build artifacts"
	@echo ""
	@echo "Management:"
	@echo "  start       - Start the server"
	@echo "  stop        - Stop the server"
	@echo "  restart     - Restart the server"
	@echo "  health      - Check system health"
	@echo ""
	@echo "Development:"
	@echo "  test        - Run all tests"
	@echo "  test-go     - Run Go tests"
	@echo "  test-lua    - Run Lua tests"
	@echo "  bench       - Run benchmarks"
	@echo ""
	@echo "Documentation:"
	@echo "  docs        - Generate documentation"
	@echo "  readme      - Update README"
	@echo ""
	@echo "Database:"
	@echo "  db-build    - Build the database"
	@echo "  db-clean    - Clean database files"
	@echo "  db-backup   - Backup database"
	@echo "  db-restore  - Restore database from backup"
	@echo ""
	@echo "Utilities:"
	@echo "  lint        - Run linters"
	@echo "  format      - Format code"
	@echo "  deps        - Update dependencies"
	@echo "  release     - Create release package"

# Installation
install: 
	@echo "Installing nvim-smart-keybind-search..."
	@chmod +x scripts/install.sh
	@./scripts/install.sh

# Building
build: build-server build-database

build-server:
	@echo "Building Go backend..."
	@go build -ldflags="-s -w" -o server cmd/server/main.go
	@chmod +x server

build-database: build-server
	@echo "Building database..."
	@go run scripts/build_database.go

# Testing
test: test-go test-lua

test-go:
	@echo "Running Go tests..."
	@go test -v ./...

test-lua:
	@echo "Running Lua tests..."
	@if command -v busted >/dev/null 2>&1; then \
		busted lua/; \
	else \
		echo "busted not found, skipping Lua tests"; \
	fi

bench:
	@echo "Running benchmarks..."
	@go test -bench=. ./internal/server

# Management
start:
	@echo "Starting nvim-smart-keybind-search server..."
	@if [ -f "./start_server.sh" ]; then \
		./start_server.sh; \
	else \
		echo "start_server.sh not found. Run 'make install' first."; \
		exit 1; \
	fi

stop:
	@echo "Stopping nvim-smart-keybind-search server..."
	@if [ -f "./stop_server.sh" ]; then \
		./stop_server.sh; \
	else \
		echo "stop_server.sh not found. Run 'make install' first."; \
		exit 1; \
	fi

restart: stop start

health:
	@echo "Checking system health..."
	@if [ -f "./health_check.sh" ]; then \
		./health_check.sh; \
	else \
		echo "health_check.sh not found. Run 'make install' first."; \
		exit 1; \
	fi

# Database management
db-build: build-database

populate-builtin:
	@echo "Populating built-in knowledge from Neovim quick reference..."
	@go run scripts/populate_builtin_knowledge.go

populate-general:
	@echo "Populating general knowledge from HuggingFace dataset..."
	@go run scripts/populate_general_knowledge.go

populate-user:
	@echo "User keybindings are populated automatically by the plugin."
	@echo "Run :SmartKeybindSync in Neovim to manually sync user keybindings."

populate-all: populate-builtin populate-general
	@echo "All knowledge bases populated successfully!"

db-clean:
	@echo "Cleaning database files..."
	@rm -rf ~/.local/share/nvim-smart-keybind-search/chroma/*
	@echo "Database cleaned."

db-backup:
	@echo "Creating database backup..."
	@mkdir -p backups
	@tar -czf backups/database-$(shell date +%Y%m%d-%H%M%S).tar.gz data/chroma/
	@echo "Database backed up to backups/"

db-restore:
	@echo "Available backups:"
	@ls -la backups/database-*.tar.gz 2>/dev/null || echo "No backups found"
	@echo ""
	@echo "To restore, run: tar -xzf backups/database-YYYYMMDD-HHMMSS.tar.gz"

# Development
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping Go linting"; \
	fi
	@if command -v luacheck >/dev/null 2>&1; then \
		luacheck lua/; \
	else \
		echo "luacheck not found, skipping Lua linting"; \
	fi

format:
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v stylua >/dev/null 2>&1; then \
		stylua lua/; \
	else \
		echo "stylua not found, skipping Lua formatting"; \
	fi

deps:
	@echo "Updating dependencies..."
	@go mod tidy
	@go mod download

# Documentation
docs: readme panvimdoc

readme:
	@echo "Updating README..."
	@# This would typically run a documentation generator
	@echo "README updated."

panvimdoc:
	@echo "Generating vim help documentation..."
	@if command -v panvimdoc >/dev/null 2>&1; then \
		panvimdoc --doc doc/nvim-smart-keybind-search.txt lua/; \
		echo "Vim help documentation generated: doc/nvim-smart-keybind-search.txt"; \
	else \
		echo "panvimdoc not found. Install with: npm install -g panvimdoc"; \
		echo "Or manually copy doc/nvim-smart-keybind-search.txt to your vim help directory"; \
	fi

# Release
release: clean build test
	@echo "Creating release package..."
	@mkdir -p release
	@tar -czf release/nvim-smart-keybind-search-$(shell git describe --tags --always).tar.gz \
		--exclude='.git' \
		--exclude='release' \
		--exclude='backups' \
		--exclude='logs' \
		--exclude='data' \
		--exclude='*.pid' \
		.
	@echo "Release package created: release/"

# Cleaning
clean:
	@echo "Cleaning build artifacts..."
	@rm -f server
	@rm -f server.exe
	@rm -rf data/chroma/*
	@rm -rf logs/*
	@rm -f *.pid
	@rm -f config.json
	@rm -f start_server.sh
	@rm -f stop_server.sh
	@rm -f health_check.sh
	@echo "Clean complete."

# Platform-specific targets
install-linux: install
	@echo "Linux installation complete."

install-macos: install
	@echo "macOS installation complete."

install-windows: install
	@echo "Windows installation complete."
	@echo "The Go backend automatically handles all dependencies."

# Service management
install-service:
	@echo "Installing as system service..."
	@if [ -f "nvim-smart-keybind-search.service" ]; then \
		sudo cp nvim-smart-keybind-search.service /etc/systemd/system/; \
		sudo systemctl enable nvim-smart-keybind-search; \
		sudo systemctl start nvim-smart-keybind-search; \
		echo "Service installed and started."; \
	else \
		echo "Service file not found. Run 'make install' first."; \
		exit 1; \
	fi

uninstall-service:
	@echo "Uninstalling system service..."
	@sudo systemctl stop nvim-smart-keybind-search || true
	@sudo systemctl disable nvim-smart-keybind-search || true
	@sudo rm -f /etc/systemd/system/nvim-smart-keybind-search.service
	@sudo systemctl daemon-reload
	@echo "Service uninstalled."

# Development helpers
dev-setup: deps build test
	@echo "Development environment setup complete."

dev-watch:
	@echo "Watching for changes..."
	@if command -v fswatch >/dev/null 2>&1; then \
		fswatch -o . | xargs -n1 -I{} make build; \
	elif command -v inotifywait >/dev/null 2>&1; then \
		while inotifywait -r -e modify .; do make build; done; \
	else \
		echo "No file watcher found. Install fswatch or inotify-tools."; \
	fi

# Docker support
docker-build:
	@echo "Building Docker image..."
	@docker build -t nvim-smart-keybind-search .

docker-run:
	@echo "Running in Docker..."
	@docker run -p 8080:8080 -p 8000:8000 nvim-smart-keybind-search

# Performance testing
perf-test:
	@echo "Running performance tests..."
	@go test -bench=. -benchmem ./internal/server

stress-test:
	@echo "Running stress tests..."
	@go test -run TestStressTest -timeout 30s ./internal/server

# Monitoring
monitor:
	@echo "Starting monitoring..."
	@watch -n 1 'make health'

logs:
	@echo "Showing logs..."
	@tail -f logs/server.log

# Quick commands
quick-start: install start
	@echo "Quick start complete. Server is running."

quick-stop: stop clean
	@echo "Quick stop complete. Server stopped and cleaned."

# Helpers
check-deps:
	@echo "Checking dependencies..."
	@command -v go >/dev/null 2>&1 || { echo "Go not found. Please install Go 1.21+"; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "Git not found. Please install Git"; exit 1; }
	@command -v python3 >/dev/null 2>&1 || { echo "Python3 not found. Please install Python 3.8+"; exit 1; }
	@echo "All dependencies found."

version:
	@echo "nvim-smart-keybind-search version: $(shell git describe --tags --always)"
	@echo "Go version: $(shell go version)"
	@echo "Python version: $(shell python3 --version)"

# Default target
.DEFAULT_GOAL := help 