# Makefile for SupplyChain Go Application

# Variables
APP_NAME := supplychain
BINARY_NAME := $(APP_NAME)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -w -s"

# Go related variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Directories
BUILD_DIR := build
DIST_DIR := dist
COVERAGE_DIR := coverage

# Docker variables
DOCKER_IMAGE := $(APP_NAME)
DOCKER_TAG := $(VERSION)
DOCKER_REGISTRY := # Set your registry here

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
.PHONY: help
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the application
.PHONY: build
build: clean
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/supplychain

## build-linux: Build for Linux (production)
.PHONY: build-linux
build-linux: clean
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/supplychain

## build-all: Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building $(BINARY_NAME) for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/supplychain
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/supplychain
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/supplychain
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/supplychain

## run: Run the application
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run ./cmd/supplychain

## test: Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

## test-race: Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -race -v ./...

## bench: Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## lint: Run linting tools
.PHONY: lint
lint: fmt vet
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## fmt: Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## vet: Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## mod-tidy: Tidy go modules
.PHONY: mod-tidy
mod-tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

## mod-download: Download go modules
.PHONY: mod-download
mod-download:
	@echo "Downloading modules..."
	$(GOMOD) download

## mod-verify: Verify go modules
.PHONY: mod-verify
mod-verify:
	@echo "Verifying modules..."
	$(GOMOD) verify

## clean: Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -rf $(COVERAGE_DIR)

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):latest

## docker-run: Run Docker container
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 $(DOCKER_IMAGE):$(DOCKER_TAG)

## docker-push: Push Docker image to registry
.PHONY: docker-push
docker-push:
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "DOCKER_REGISTRY not set. Please set it in the Makefile or environment."; \
		exit 1; \
	fi
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest

## install: Install the application
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

## uninstall: Uninstall the application
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f /usr/local/bin/$(BINARY_NAME)

## deps: Install development dependencies
.PHONY: deps
deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

## security: Run security checks
.PHONY: security
security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

## release: Create a release build
.PHONY: release
release: clean test lint build-all
	@echo "Creating release $(VERSION)..."
	@mkdir -p $(DIST_DIR)
	@for binary in $(BUILD_DIR)/*; do \
		if [ -f "$$binary" ]; then \
			tar -czf $(DIST_DIR)/$$(basename $$binary).tar.gz -C $(BUILD_DIR) $$(basename $$binary); \
		fi \
	done
	@echo "Release artifacts created in $(DIST_DIR)/"

## ci: Run CI pipeline (test, lint, build)
.PHONY: ci
ci: mod-download test-race lint security build

## dev: Development setup
.PHONY: dev
dev: deps mod-tidy fmt test

## version: Show version information
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

## check: Run all checks (fmt, vet, test, lint)
.PHONY: check
check: fmt vet test lint

.PHONY: all
all: clean deps check build