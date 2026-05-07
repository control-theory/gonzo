# Gonzo - Makefile  
# Modern TUI dashboard for OTLP log analysis

# Project configuration
BINARY_NAME := gonzo
CMD_DIR := ./cmd/gonzo
BUILD_DIR := ./build
DIST_DIR := ./dist

# Version and build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go configuration
GO := go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

# Build flags
GO_VERSION := $(shell go version | cut -d' ' -f3)
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)"
BUILD_FLAGS := -trimpath $(LDFLAGS)

# Colors for pretty output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
PURPLE := \033[0;35m
CYAN := \033[0;36m
NC := \033[0m # No Color

# Code signing (macOS) — set via environment or here
CODESIGN_IDENTITY ?= -

.PHONY: all build build-tui clean clean-all test test-race test-integration \
        fmt vet lint install uninstall run demo \
        deps deps-update deps-tidy cross-build release help dev ci info \
        web-deps web-build web-dev web-clean build-signed

# Default target
all: clean build

# Help target
help: ## Show this help message
	@echo "$(CYAN)Gonzo Build System$(NC)"
	@echo "==================="
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make build              # Build the TUI binary"
	@echo "  make demo              # Demo with sample data"
	@echo "  make cross-build       # Build for all platforms"
	@echo "  make install           # Install to GOPATH/bin"

# Web dashboard targets
web-deps: ## Install web dashboard dependencies
	@echo "$(BLUE)Installing web dependencies...$(NC)"
	@cd web && rm -rf node_modules && npm ci
	@echo "$(GREEN)✓ Web dependencies installed$(NC)"

web-build: web-deps ## Build web dashboard
	@echo "$(BLUE)Building web dashboard...$(NC)"
	@cd web && npm run build
	@echo "$(GREEN)✓ Web dashboard built$(NC)"

web-dev: web-deps ## Start web dev server with API proxy
	@echo "$(BLUE)Starting web dev server...$(NC)"
	@cd web && npm run dev

web-clean: ## Clean web build artifacts
	@echo "$(BLUE)Cleaning web artifacts...$(NC)"
	@rm -rf web/dist web/node_modules
	@echo "$(GREEN)✓ Web artifacts cleaned$(NC)"

# Ensure web/dist exists for go:embed (stub if not built)
ensure-web-dist:
	@mkdir -p web/dist
	@test -f web/dist/index.html || echo '<!DOCTYPE html><html><body></body></html>' > web/dist/index.html

# Build targets
build: deps web-build ## Build binary with embedded web dashboard
	@echo "$(BLUE)Building Gonzo with web dashboard...$(NC)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "$(GREEN)✓ Built $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-signed: build ## Build and codesign for macOS (set CODESIGN_IDENTITY for distribution)
	@echo "$(BLUE)Signing $(BUILD_DIR)/$(BINARY_NAME)...$(NC)"
	@codesign --force --options runtime --sign "$(CODESIGN_IDENTITY)" --timestamp $(BUILD_DIR)/$(BINARY_NAME)
	@echo "$(GREEN)✓ Signed $(BUILD_DIR)/$(BINARY_NAME)$(NC)"
	@codesign -dvv $(BUILD_DIR)/$(BINARY_NAME) 2>&1 | head -5

build-tui: deps ensure-web-dist ## Build TUI-only binary (placeholder web dashboard)
	@echo "$(BLUE)Building TUI-only version...$(NC)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "$(GREEN)✓ Built $(BUILD_DIR)/$(BINARY_NAME) (TUI only)$(NC)"

# Cross-platform builds
cross-build: clean deps web-build ## Build for multiple platforms
	@echo "$(BLUE)Building for multiple platforms...$(NC)"
	@mkdir -p $(DIST_DIR)
	
	# Linux amd64
	@echo "$(YELLOW)Building for linux/amd64...$(NC)"
	@mkdir -p $(DIST_DIR)/linux-amd64
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 \
		$(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/linux-amd64/$(BINARY_NAME) $(CMD_DIR)
	
	# macOS amd64
	@echo "$(YELLOW)Building for darwin/amd64...$(NC)"
	@mkdir -p $(DIST_DIR)/darwin-amd64
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=amd64 \
		$(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/darwin-amd64/$(BINARY_NAME) $(CMD_DIR)
	
	# macOS arm64
	@echo "$(YELLOW)Building for darwin/arm64...$(NC)"
	@mkdir -p $(DIST_DIR)/darwin-arm64
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=darwin GOARCH=arm64 \
		$(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/darwin-arm64/$(BINARY_NAME) $(CMD_DIR)
	
	# Windows amd64
	@echo "$(YELLOW)Building for windows/amd64...$(NC)"
	@mkdir -p $(DIST_DIR)/windows-amd64
	@CGO_ENABLED=$(CGO_ENABLED) GOOS=windows GOARCH=amd64 \
		$(GO) build $(BUILD_FLAGS) -o $(DIST_DIR)/windows-amd64/$(BINARY_NAME).exe $(CMD_DIR)
	
	@echo "$(GREEN)✓ Cross-platform builds complete in $(DIST_DIR)/$(NC)"

# Dependency management
deps: ## Download and verify dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@$(GO) mod download
	@$(GO) mod verify

deps-update: ## Update all dependencies
	@echo "$(BLUE)Updating dependencies...$(NC)"
	@$(GO) get -u ./...
	@$(GO) mod tidy

deps-tidy: ## Clean up dependencies
	@echo "$(BLUE)Tidying dependencies...$(NC)"
	@$(GO) mod tidy

# Testing
test: deps ensure-web-dist ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	@$(GO) test -v ./...

test-race: deps ensure-web-dist ## Run tests with race detection
	@echo "$(BLUE)Running tests with race detection...$(NC)"
	@$(GO) test -race -v ./...

test-integration: build ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@if [ -f test.json ]; then \
		echo "Testing TUI mode..."; \
		if head -n 5 test.json | \
		((command -v timeout >/dev/null 2>&1 && timeout 3s $(BUILD_DIR)/$(BINARY_NAME) --update-interval=1s) || \
		 (command -v gtimeout >/dev/null 2>&1 && gtimeout 3s $(BUILD_DIR)/$(BINARY_NAME) --update-interval=1s) || \
		 ($(BUILD_DIR)/$(BINARY_NAME) --update-interval=1s & PID=$$!; sleep 3; kill $$PID 2>/dev/null || true)); then \
			echo "$(GREEN)✓ TUI mode test completed$(NC)"; \
		else \
			echo "$(YELLOW)⚠ Integration test had issues but continued$(NC)"; \
		fi; \
	else \
		echo "$(YELLOW)⚠ test.json not found, skipping integration tests$(NC)"; \
		echo "$(CYAN)Creating sample test data...$(NC)"; \
		echo '{"timestamp":"2024-08-05T10:30:45Z","level":"INFO","message":"Test log entry","service":"test"}' | \
		$(BUILD_DIR)/$(BINARY_NAME) --update-interval=1s > /dev/null 2>&1 && \
		echo "$(GREEN)✓ Basic functionality test passed$(NC)" || \
		echo "$(RED)✗ Basic functionality test failed$(NC)"; \
	fi

# Code quality
fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	@$(GO) fmt ./...

vet: ensure-web-dist ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	@$(GO) vet ./...

lint: ## Run linter (requires golangci-lint)
	@echo "$(BLUE)Running linter...$(NC)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)golangci-lint not found. Installing...$(NC)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run

# Installation
install: build ## Install binary to $GOPATH/bin
	@echo "$(BLUE)Installing binary...$(NC)"
	@$(GO) install $(BUILD_FLAGS) $(CMD_DIR)
	@echo "$(GREEN)✓ Installed $(BINARY_NAME) to $(shell go env GOPATH)/bin$(NC)"

uninstall: ## Remove installed binary
	@echo "$(BLUE)Uninstalling binary...$(NC)"
	@rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ Uninstalled binary$(NC)"

# Development helpers
run: build ## Run TUI version with sample data (requires TTY)
	@echo "$(BLUE)Running TUI version...$(NC)"
	@echo "$(YELLOW)Note: This requires a real terminal (TTY)$(NC)"
	@if [ -f test.json ]; then \
		head -n 20 test.json | $(BUILD_DIR)/$(BINARY_NAME) --update-interval=2s; \
	else \
		echo "$(RED)test.json not found. Please ensure you have sample OTLP data.$(NC)"; \
	fi

demo: build ## Demo TUI version (shows instructions)
	@echo "$(PURPLE)🎯 TUI Log Analyzer Dashboard Demo$(NC)"
	@echo "=================================="
	@echo ""
	@echo "$(YELLOW)To run the interactive TUI dashboard:$(NC)"
	@echo ""
	@echo "  $(GREEN)make run$(NC)     # Run with test.json data"
	@echo "  $(GREEN)./$(BUILD_DIR)/$(BINARY_NAME)$(NC)  # Run and pipe your own data"
	@echo ""
	@echo "$(CYAN)Example with live data:$(NC)"
	@echo "  tail -f /var/log/app.log | ./$(BUILD_DIR)/$(BINARY_NAME)"
	@echo "  kubectl logs -f deployment/my-app | ./$(BUILD_DIR)/$(BINARY_NAME)"
	@echo ""
	@if [ -x ./test_tui.sh ]; then \
		./test_tui.sh; \
	else \
		echo "$(YELLOW)Note: test_tui.sh not found or not executable$(NC)"; \
		echo "$(CYAN)Try running: ./$(BUILD_DIR)/$(BINARY_NAME) --test-mode$(NC)"; \
	fi

# Release management
release: clean cross-build test ## Create a release build
	@echo "$(BLUE)Creating release archives...$(NC)"
	@cd $(DIST_DIR) && for dir in */; do \
		platform=$$(basename "$$dir"); \
		echo "$(YELLOW)Creating archive for $$platform...$(NC)"; \
		tar -czf "$(BINARY_NAME)-$(VERSION)-$$platform.tar.gz" -C "$$dir" .; \
	done
	@echo "$(GREEN)✓ Release archives created in $(DIST_DIR)/$(NC)"
	@ls -la $(DIST_DIR)/*.tar.gz

# Cleanup
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -rf web/dist
	@rm -f $(BINARY_NAME)
	@echo "$(GREEN)✓ Cleaned$(NC)"

clean-all: clean ## Clean build artifacts and test files
	@echo "$(BLUE)Cleaning all artifacts and test files...$(NC)"
	@rm -f test_*.sh test_*.go
	@echo "$(GREEN)✓ Cleaned all$(NC)"

# Development workflow
dev: fmt vet test build ## Full development workflow (format, vet, test, build)
	@echo "$(GREEN)✓ Development workflow complete$(NC)"

# CI/CD workflow  
ci: deps fmt vet lint test build test-integration ## CI/CD pipeline
	@echo "$(GREEN)✓ CI pipeline complete$(NC)"

# Show build info
info: ## Show build information
	@echo "$(CYAN)Build Information$(NC)"
	@echo "================="
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(shell go version)"
	@echo "Platform:   $(GOOS)/$(GOARCH)"
	@echo "CGO:        $(CGO_ENABLED)"
