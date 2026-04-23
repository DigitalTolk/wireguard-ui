APP_NAME    := wg-ui
GOBIN       := $(shell go env GOPATH)/bin
GO_PACKAGES := $(shell go list ./... | grep -v 'wireguard-ui$$' | grep -v node_modules)

VERSION     ?= dev
GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "N/A")
BUILD_TIME  := $(shell date -u '+%Y-%m-%d %H:%M:%S')
LDFLAGS     := -s -w -X 'main.appVersion=$(VERSION)' -X 'main.buildTime=$(BUILD_TIME)' -X 'main.gitCommit=$(GIT_COMMIT)'

.PHONY: help build build-frontend build-backend test test-verbose coverage lint lint-go lint-frontend fmt vet clean dev

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## ---- Build ----

build: build-frontend build-backend ## Build everything (frontend + Go binary)

build-frontend: ## Build the React frontend
	npm ci && npm run build

build-backend: ## Build the Go binary
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(APP_NAME) .

## ---- Test ----

test: test-go test-frontend ## Run all tests (Go + frontend) with coverage

test-go: ## Run Go tests with coverage
	@echo "=== Go Tests ==="
	@go test $(GO_PACKAGES) -coverprofile=coverage.out -count=1 -timeout 180s
	@echo ""
	@go tool cover -func=coverage.out | grep "^total:" | awk '{print "Go coverage: " $$3}'
	@rm -f coverage.out

test-frontend: ## Run frontend tests with coverage
	@echo ""
	@echo "=== Frontend Tests ==="
	@npx vitest run --coverage 2>&1 | grep -E "Tests|Statements|Lines"

test-verbose: ## Run all Go tests with verbose output
	go test $(GO_PACKAGES) -v -count=1 -timeout 180s

test-race: ## Run tests with race detector
	go test $(GO_PACKAGES) -race -count=1 -timeout 180s

coverage: ## Run Go coverage with detailed report
	go test $(GO_PACKAGES) -coverprofile=coverage.out -timeout 180s
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "To view HTML report: go tool cover -html=coverage.out"

coverage-html: coverage ## Open Go coverage report in browser
	go tool cover -html=coverage.out

## ---- Lint ----

lint: lint-go lint-frontend ## Run all linters

lint-go: ## Run Go linters (golangci-lint)
	$(GOBIN)/golangci-lint run --timeout 5m

lint-frontend: ## Run frontend linter (eslint)
	npm run lint

## ---- Format & Vet ----

fmt: ## Format Go code
	gofmt -w -s .
	goimports -w .

vet: ## Run go vet
	go vet $(GO_PACKAGES)

## ---- Development ----

dev: ## Start the Go app for development
	go run -ldflags="$(LDFLAGS)" .

dev-frontend: ## Start the frontend dev server with hot reload
	npm run dev

## ---- Dependencies ----

deps: ## Install/update Go dependencies
	go mod tidy
	go mod download

deps-frontend: ## Install frontend dependencies
	npm ci

## ---- Clean ----

clean: ## Remove build artifacts
	rm -f $(APP_NAME) coverage.out
	rm -rf assets/*.html assets/assets/

## ---- Docker ----

docker-build: ## Build Docker image
	docker build -t wireguard-ui:$(VERSION) .

docker-run: ## Run Docker container (requires NET_ADMIN and host network)
	docker run --rm -it \
		--cap-add NET_ADMIN \
		--network host \
		-v ./db:/app/db \
		-v /etc/wireguard:/etc/wireguard \
		wireguard-ui:$(VERSION)
