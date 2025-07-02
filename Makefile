# Makefile for Shade project
# Handles building, cleaning, and compiling both the Go backend and Chrome extension

# Variables
BINARY_NAME=shade
EXTENSION_DIR=extension
BACKEND_DIR=backend
DEV_FILE=dev.yml

.PHONY: all backend extension clean install-deps

# Default target: build everything
all: extension dev

# Build the Go backend
backend:
	@echo "Building Go backend..."
	cd backend/ && go build -o $(BINARY_NAME) ./cmd/...
	@echo "Backend built successfully: $(BINARY_NAME)"

dev:
	@echo "Running Go backend in development mode..."
	cd $(BACKEND_DIR) && go run ./cmd/... -log=debug -config=$(DEV_FILE)

# Build the Chrome extension
extension: install-deps
	@echo "Building Chrome extension..."
	cd $(EXTENSION_DIR) && npm run build
	@echo "Extension built successfully"

# Install dependencies for the extension
install-deps:
	@echo "Installing extension dependencies..."
	cd $(EXTENSION_DIR) && npm install
	@echo "Dependencies installed successfully"

# Clean up build artifacts
clean:
	@echo "Cleaning up build artifacts..."
	rm -f $(BINARY_NAME)
	rm -rf $(EXTENSION_DIR)/dist
	@echo "Clean complete"

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Build both backend and extension (default)"
	@echo "  backend      - Build only the Go backend"
	@echo "  extension    - Build only the Chrome extension"
	@echo "  install-deps - Install dependencies for the extension"
	@echo "  clean        - Remove all build artifacts"
	@echo "  help         - Show this help message"
