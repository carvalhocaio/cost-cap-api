.PHONY: help deps tidy build run test cover lint lint-fix format format-check audit ci check clean

NAME ?= cost-cap-api
PKG := $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(PKG)/internal/app.version=$(VERSION)

help: ## Lists all available Makefile commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

deps: ## Downloads module dependencies
	go mod download

tidy: ## Adds missing and removes unused module dependencies
	go mod tidy

build: ## Builds the binary into bin/
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(NAME) ./cmd/$(NAME)

run: ## Runs the application: make run ARGS="-name Gopher"
	go run -ldflags "$(LDFLAGS)" ./cmd/$(NAME) $(ARGS)

test: ## Runs the test suite with the race detector
	go test -race ./...

cover: ## Runs tests and generates a coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint: ## Checks code with golangci-lint
	golangci-lint run

lint-fix: ## Automatically fixes golangci-lint issues
	golangci-lint run --fix

format: ## Formats code with golangci-lint (gofumpt + goimports)
	golangci-lint fmt

format-check: ## Verifies formatting without modifying files
	@out="$$(golangci-lint fmt --diff)"; \
	if [ -n "$$out" ]; then echo "$$out"; exit 1; fi

audit: ## Audits dependencies for known vulnerabilities with govulncheck
	go tool govulncheck ./...

ci: lint format-check audit test ## Runs full verification pipeline locally

check: ci ## Alias for ci

clean: ## Cleans build artifacts and caches
	rm -rf bin coverage.out
	go clean -testcache
