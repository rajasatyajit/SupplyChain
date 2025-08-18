# Makefile for SupplyChain Microservice

# Variables
APP_NAME := supplychain
BINARY_NAME := $(APP_NAME)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -w -s -extldflags '-static'"

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
REPORTS_DIR := reports

# Docker variables
DOCKER_NAMESPACE ?= satyajitr13
DOCKER_IMAGE ?= $(APP_NAME)
DOCKER_TAG := $(VERSION)
DOCKER_REGISTRY := # Set your registry here (e.g., your-registry.com/namespace)

# Test variables
TEST_TIMEOUT := 30s
INTEGRATION_TEST_TIMEOUT := 60s

# Linting variables
GOLANGCI_LINT_VERSION := v1.55.2

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
.PHONY: help
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the application (optimized)
.PHONY: build
build: clean
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/supplychain
	@echo "Binary size: $$(du -h $(BUILD_DIR)/$(BINARY_NAME) | cut -f1)"

## build-debug: Build with debug symbols
.PHONY: build-debug
build-debug: clean
	@echo "Building $(BINARY_NAME) with debug symbols..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug ./cmd/supplychain

## build-linux: Build for Linux (production)
.PHONY: build-linux
build-linux: clean
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/supplychain
	@echo "Linux binary size: $$(du -h $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 | cut -f1)"

## build-all: Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building $(BINARY_NAME) for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/supplychain
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/supplychain
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/supplychain
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/supplychain
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/supplychain
	@echo "Build sizes:"
	@du -h $(BUILD_DIR)/* | sort -h

## strip: Strip debug symbols from binary
.PHONY: strip
strip:
	@if [ -f $(BUILD_DIR)/$(BINARY_NAME) ]; then \
		echo "Stripping debug symbols..."; \
		strip $(BUILD_DIR)/$(BINARY_NAME); \
		echo "Stripped binary size: $$(du -h $(BUILD_DIR)/$(BINARY_NAME) | cut -f1)"; \
	else \
		echo "Binary not found. Run 'make build' first."; \
	fi

## run: Run the application
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	$(GOCMD) run ./cmd/supplychain

## test: Run unit tests
.PHONY: test
test:
	@echo "Running unit tests..."
	@mkdir -p $(REPORTS_DIR)
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) -coverprofile=$(REPORTS_DIR)/coverage.out ./...
	@echo "Unit tests completed"

## test-integration: Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@mkdir -p $(REPORTS_DIR)
	$(GOTEST) -v -timeout=$(INTEGRATION_TEST_TIMEOUT) -tags=integration ./test/integration/...
	@echo "Integration tests completed"

## test-coverage: Run tests with detailed coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage analysis..."
	@mkdir -p $(COVERAGE_DIR) $(REPORTS_DIR)
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) -coverpkg=./... -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out > $(REPORTS_DIR)/coverage.txt
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"
	@echo "Coverage summary: $(REPORTS_DIR)/coverage.txt"

## test-cover-check: Enforce minimum coverage threshold (default 85%)
.PHONY: test-cover-check
test-cover-check: test-coverage
	@echo "Verifying coverage threshold..."
	@COVERAGE_THRESHOLD=${COVERAGE_THRESHOLD:=85} bash scripts/check_coverage.sh $$COVERAGE_THRESHOLD $(COVERAGE_DIR)/coverage.out

## test-race: Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -race -v -timeout=$(TEST_TIMEOUT) ./...

## test-all: Run all tests (unit, integration, race)
.PHONY: test-all
test-all: test test-integration test-race
	@echo "All tests completed"

## bench: Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	@mkdir -p $(REPORTS_DIR)
	$(GOTEST) -bench=. -benchmem -benchtime=5s ./... | tee $(REPORTS_DIR)/benchmark.txt
	@echo "Benchmark results: $(REPORTS_DIR)/benchmark.txt"

## lint: Run comprehensive linting
.PHONY: lint
lint: fmt vet lint-golangci
	@echo "All linting completed"

## lint-install: Install linting tools
.PHONY: lint-install
lint-install:
	@echo "Installing linting tools..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
	fi
	@echo "Linting tools installed"

## lint-golangci: Run golangci-lint
.PHONY: lint-golangci
lint-golangci:
	@echo "Running golangci-lint..."
	@mkdir -p $(REPORTS_DIR)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --out-format=colored-line-number,checkstyle:$(REPORTS_DIR)/golangci-lint.xml ./...; \
	else \
		echo "golangci-lint not installed. Run 'make lint-install' first"; \
		exit 1; \
	fi

## lint-fix: Run linting with auto-fix
.PHONY: lint-fix
lint-fix: fmt
	@echo "Running golangci-lint with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix ./...; \
	else \
		echo "golangci-lint not installed. Run 'make lint-install' first"; \
		exit 1; \
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
	rm -rf $(REPORTS_DIR)
	rm -rf vendor/

## docker-build: Build optimized Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	@echo "Image size: $$(docker images $(DOCKER_IMAGE):$(DOCKER_TAG) --format 'table {{.Size}}')"

## docker-build-multi: Build multi-architecture Docker image
.PHONY: docker-build-multi
docker-build-multi:
	@echo "Building multi-architecture Docker image..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		--push .

## docker-run: Run Docker container locally
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run --rm \
		-p 8080:8080 \
		-p 9090:9090 \
		-e LOG_LEVEL=debug \
		--name $(APP_NAME)-dev \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

## docker-run-prod: Run Docker container in production mode
.PHONY: docker-run-prod
docker-run-prod:
	@echo "Running Docker container in production mode..."
	docker run -d \
		-p 8080:8080 \
		-p 9090:9090 \
		-e LOG_LEVEL=info \
		-e LOG_FORMAT=json \
		-e METRICS_ENABLED=true \
		--restart unless-stopped \
		--name $(APP_NAME)-prod \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

## docker-push: Push Docker image to registry
.PHONY: docker-push
docker-push:
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "DOCKER_REGISTRY not set. Set with: make docker-push DOCKER_REGISTRY=your-registry.com"; \
		exit 1; \
	fi
	@echo "Pushing to registry: $(DOCKER_REGISTRY)"
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):latest
	@echo "Images pushed successfully"

## docker-push-hub: Push Docker image to Docker Hub (namespace $(DOCKER_NAMESPACE))
.PHONY: docker-push-hub
docker-push-hub:
	@echo "Pushing image to Docker Hub: $(DOCKER_NAMESPACE)/$(DOCKER_IMAGE):$(DOCKER_TAG)"
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_NAMESPACE)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_NAMESPACE)/$(DOCKER_IMAGE):latest
	docker push $(DOCKER_NAMESPACE)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_NAMESPACE)/$(DOCKER_IMAGE):latest
	@echo "Docker Hub push complete"

## docker-scan: Scan Docker image for vulnerabilities
.PHONY: docker-scan
docker-scan:
	@echo "Scanning Docker image for vulnerabilities..."
	@if command -v trivy >/dev/null 2>&1; then \
		trivy image $(DOCKER_IMAGE):$(DOCKER_TAG); \
	else \
		echo "Trivy not installed. Using docker scout if available..."; \
		docker scout cves $(DOCKER_IMAGE):$(DOCKER_TAG) || echo "No vulnerability scanner available"; \
	fi

## k8s-deploy: Deploy to Kubernetes
.PHONY: k8s-deploy
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "DOCKER_REGISTRY not set. Set with: make k8s-deploy DOCKER_REGISTRY=your-registry.com"; \
		exit 1; \
	fi
	@sed 's|supplychain:latest|$(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)|g' deployments/kubernetes/deployment.yaml | kubectl apply -f -
	kubectl apply -f deployments/kubernetes/
	@echo "Deployment completed"

## k8s-status: Check Kubernetes deployment status
.PHONY: k8s-status
k8s-status:
	@echo "Checking deployment status..."
	kubectl get pods -l app=$(APP_NAME)
	kubectl get services -l app=$(APP_NAME)
	kubectl get ingress -l app=$(APP_NAME)

## k8s-logs: View Kubernetes logs
.PHONY: k8s-logs
k8s-logs:
	@echo "Viewing application logs..."
	kubectl logs -l app=$(APP_NAME) --tail=100 -f

## k8s-delete: Delete Kubernetes deployment
.PHONY: k8s-delete
k8s-delete:
	@echo "Deleting Kubernetes deployment..."
	kubectl delete -f deployments/kubernetes/
	@echo "Deployment deleted"

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
deps: lint-install
	@echo "Installing development dependencies..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@echo "Development dependencies installed"

## security: Run comprehensive security checks
.PHONY: security
security:
	@echo "Running security analysis..."
	@mkdir -p $(REPORTS_DIR)
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -fmt=json -out=$(REPORTS_DIR)/gosec.json ./...; \
		gosec -fmt=text ./...; \
	else \
		echo "gosec not installed. Run 'make deps' first"; \
		exit 1; \
	fi

## vuln: Check for known vulnerabilities
.PHONY: vuln
vuln:
	@echo "Checking for vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi

## complexity: Check code complexity
.PHONY: complexity
complexity:
	@echo "Analyzing code complexity..."
	@mkdir -p $(REPORTS_DIR)
	@if command -v gocyclo >/dev/null 2>&1; then \
		gocyclo -over 10 . | tee $(REPORTS_DIR)/complexity.txt; \
	else \
		echo "gocyclo not installed. Run 'make deps' first"; \
		exit 1; \
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

## ci: Run full CI pipeline
.PHONY: ci
ci: mod-download test-all lint security vuln complexity build
	@echo "CI pipeline completed successfully"

## pre-commit: Run pre-commit checks
.PHONY: pre-commit
pre-commit: fmt vet lint-fix test
	@echo "Pre-commit checks completed"

## quality: Run comprehensive quality checks
.PHONY: quality
quality: test-coverage lint security vuln complexity
	@echo "Quality analysis completed"

## microservice: Build optimized microservice
.PHONY: microservice
microservice: clean build-linux strip
	@echo "Microservice build completed"
	@echo "Final binary: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"
	@echo "Size: $$(du -h $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 | cut -f1)"

## smoke: Run smoke checks against local service
.PHONY: smoke
smoke:
	@bash scripts/smoke.sh http://localhost:8080/health
	@bash scripts/smoke.sh http://localhost:8080/v1/health/ready

## dev: Development setup
.PHONY: dev
dev: deps mod-tidy fmt test
	@echo "Development environment ready"

## version: Show version information
.PHONY: version
version:
	@echo "Application: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $$(go version)"

## check: Run all checks (fmt, vet, test, lint)
.PHONY: check
check: fmt vet test lint
	@echo "All checks completed"

## validate: Validate project structure and dependencies
.PHONY: validate
validate:
	@echo "Validating project structure..."
	@test -f go.mod || (echo "go.mod not found" && exit 1)
	@test -f cmd/supplychain/main.go || (echo "main.go not found" && exit 1)
	@test -f Dockerfile || (echo "Dockerfile not found" && exit 1)
	@echo "Project structure validation passed"

.PHONY: all
all: clean deps validate check quality build
	@echo "Full build pipeline completed"

## Performance and Profiling Targets

## profile-cpu: Run CPU profiling
.PHONY: profile-cpu
profile-cpu:
	@echo "Running CPU profiling..."
	@mkdir -p $(REPORTS_DIR)
	go test -cpuprofile=$(REPORTS_DIR)/cpu.prof -bench=. ./...
	@echo "CPU profile: $(REPORTS_DIR)/cpu.prof"
	@echo "View with: go tool pprof $(REPORTS_DIR)/cpu.prof"

## profile-mem: Run memory profiling
.PHONY: profile-mem
profile-mem:
	@echo "Running memory profiling..."
	@mkdir -p $(REPORTS_DIR)
	go test -memprofile=$(REPORTS_DIR)/mem.prof -bench=. ./...
	@echo "Memory profile: $(REPORTS_DIR)/mem.prof"
	@echo "View with: go tool pprof $(REPORTS_DIR)/mem.prof"

## load-test: Run basic load test (requires binary)
.PHONY: load-test
load-test:
	@if [ ! -f $(BUILD_DIR)/$(BINARY_NAME) ]; then \
		echo "Binary not found. Run 'make build' first."; \
		exit 1; \
	fi
	@echo "Starting application for load testing..."
	@$(BUILD_DIR)/$(BINARY_NAME) &
	@APP_PID=$$!; \
	sleep 2; \
	echo "Running load test..."; \
	if command -v hey >/dev/null 2>&1; then \
		hey -n 1000 -c 10 http://localhost:8080/health; \
	elif command -v ab >/dev/null 2>&1; then \
		ab -n 1000 -c 10 http://localhost:8080/health; \
	else \
		echo "No load testing tool found. Install 'hey' or 'ab'"; \
	fi; \
	kill $$APP_PID 2>/dev/null || true

## Deployment and Operations Targets

## deploy-staging: Deploy to staging environment
.PHONY: deploy-staging
deploy-staging: docker-build docker-push
	@echo "Deploying to staging..."
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "DOCKER_REGISTRY not set"; \
		exit 1; \
	fi
	@kubectl config use-context staging 2>/dev/null || echo "Staging context not found"
	@sed 's|supplychain:latest|$(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)|g' deployments/kubernetes/deployment.yaml | \
		sed 's|replicas: 3|replicas: 2|g' | \
		kubectl apply -f -
	@echo "Staging deployment completed"

## deploy-prod: Deploy to production environment
.PHONY: deploy-prod
deploy-prod: docker-build docker-scan docker-push
	@echo "Deploying to production..."
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "DOCKER_REGISTRY not set"; \
		exit 1; \
	fi
	@echo "WARNING: This will deploy to production. Continue? [y/N]"
	@read -r REPLY; \
	if [ "$$REPLY" = "y" ] || [ "$$REPLY" = "Y" ]; then \
		kubectl config use-context production 2>/dev/null || echo "Production context not found"; \
		sed 's|supplychain:latest|$(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)|g' deployments/kubernetes/deployment.yaml | \
			kubectl apply -f -; \
		echo "Production deployment completed"; \
	else \
		echo "Deployment cancelled"; \
	fi

## rollback: Rollback Kubernetes deployment
.PHONY: rollback
rollback:
	@echo "Rolling back deployment..."
	kubectl rollout undo deployment/$(APP_NAME)
	kubectl rollout status deployment/$(APP_NAME)
	@echo "Rollback completed"

## Health and Monitoring Targets

## health-check: Check application health
.PHONY: health-check
health-check:
	@echo "Checking application health..."
	@curl -f http://localhost:8080/health || echo "Health check failed"
	@curl -f http://localhost:8080/v1/health/ready || echo "Readiness check failed"
	@curl -f http://localhost:8080/v1/health/live || echo "Liveness check failed"

## metrics: View metrics
.PHONY: metrics
metrics:
	@echo "Fetching metrics..."
	@curl -s http://localhost:9090/metrics | head -20

## logs: View application logs (local)
.PHONY: logs
logs:
	@if [ -f $(BUILD_DIR)/$(BINARY_NAME) ]; then \
		echo "Starting application to view logs..."; \
		$(BUILD_DIR)/$(BINARY_NAME); \
	else \
		echo "Binary not found. Run 'make build' first."; \
	fi

## Maintenance Targets

## update-deps: Update all dependencies
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "Dependencies updated"

## audit: Run security audit
.PHONY: audit
audit: security vuln
	@echo "Security audit completed"

## size-report: Generate binary size report
.PHONY: size-report
size-report: build
	@echo "Binary size analysis:"
	@echo "====================="
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)
	@echo ""
	@echo "Stripped size:"
	@strip $(BUILD_DIR)/$(BINARY_NAME) 2>/dev/null || true
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)
	@echo ""
	@echo "Sections:"
	@size $(BUILD_DIR)/$(BINARY_NAME) 2>/dev/null || echo "size command not available"

## generate-docs: Generate documentation
.PHONY: generate-docs
generate-docs:
	@echo "Generating documentation..."
	@mkdir -p docs/generated
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Generating Go documentation..."; \
		godoc -http=:6060 & \
		GODOC_PID=$$!; \
		sleep 2; \
		curl -s http://localhost:6060/pkg/github.com/rajasatyajit/SupplyChain/ > docs/generated/godoc.html; \
		kill $$GODOC_PID 2>/dev/null || true; \
	fi
	@echo "Documentation generated in docs/generated/"

## Utility Targets

## env-check: Check environment setup
.PHONY: env-check
env-check:
	@echo "Environment Check:"
	@echo "=================="
	@echo "Go version: $$(go version)"
	@echo "Docker: $$(docker --version 2>/dev/null || echo 'Not installed')"
	@echo "Kubectl: $$(kubectl version --client --short 2>/dev/null || echo 'Not installed')"
	@echo "Git: $$(git --version)"
	@echo ""
	@echo "Required tools:"
	@echo "- golangci-lint: $$(golangci-lint --version 2>/dev/null || echo 'Not installed - run make lint-install')"
	@echo "- gosec: $$(gosec -version 2>/dev/null || echo 'Not installed - run make deps')"
	@echo "- hey (load testing): $$(hey -version 2>/dev/null || echo 'Not installed - optional')"

## clean-all: Deep clean (including Docker images)
.PHONY: clean-all
clean-all: clean
	@echo "Deep cleaning..."
	@docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true
	@docker rmi $(DOCKER_IMAGE):latest 2>/dev/null || true
	@docker system prune -f
	@echo "Deep clean completed"

## Full Pipeline Targets

## pipeline-dev: Full development pipeline
.PHONY: pipeline-dev
pipeline-dev: clean env-check deps validate fmt vet test lint build
	@echo "Development pipeline completed successfully"

## pipeline-ci: Full CI pipeline
.PHONY: pipeline-ci
pipeline-ci: clean validate mod-download test-all lint security vuln complexity build docker-build
	@echo "CI pipeline completed successfully"

## pipeline-cd: Full CD pipeline
.PHONY: pipeline-cd
pipeline-cd: pipeline-ci docker-scan docker-push k8s-deploy
	@echo "CD pipeline completed successfully"