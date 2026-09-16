.DEFAULT_GOAL := help

## Tool Versions

# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2

# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.7.0

## Shared development tools

include bin/dev-tools.mk
include $(call dev-tools-module,tag)
include $(call dev-tools-module,help)

## Project-local tools

GOLANGCI_LINT := bin/golangci-lint

## Build Configuration

BINARY ?= kumbuka-cli
COMMAND ?= ./cmd/kumbuka-cli
BUILD_VERSION ?= dev
LDFLAGS ?= -s -w -X main.Version=$(BUILD_VERSION)


##@ Development

.PHONY: download
download: dev-tools ## Download Go and development dependencies.
	go mod download

.PHONY: build
build: ## Build the standalone Kumbuka CLI.
	go build -ldflags="$(LDFLAGS)" -o $(BINARY) $(COMMAND)

.PHONY: vet
vet: ## Run Go static analysis.
	go vet ./...

.PHONY: test
test: vet ## Run unit tests.
	go test -covermode=set -timeout=3m ./...

.PHONY: test-fresh
test-fresh: vet ## Run unit tests without the Go test cache.
	go test -covermode=set -count=1 -timeout=3m ./...

.PHONY: test-race
test-race: vet ## Run unit tests with the race detector.
	go test -race -count=1 -timeout=3m ./...

.PHONY: cover
cover: ## Display Go test coverage.
	go test -coverprofile=coverage.out -covermode=set -count=1 -timeout=3m ./...
	go tool cover -html=coverage.out

.PHONY: clean
clean: ## Clean generated files.
	rm -f $(BINARY) coverage.out coverage.html


##@ Formatting

.PHONY: fmt
fmt: ## Format Go code.
	go fmt ./...


##@ Linting

.PHONY: lint
lint: lint-go ## Run all linters.

.PHONY: lint-go
lint-go: golangci-lint ## Run golangci-lint.
	$(call run-tool,$(GOLANGCI_LINT),run)

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint and apply fixes.
	$(call run-tool,$(GOLANGCI_LINT),run --fix)


##@ Dependencies

.PHONY: dev-tools
dev-tools: $(DEV_TAG) $(MAKE_HELP) $(GO_INSTALL_TOOL) ## Download pinned development tools.

.PHONY: golangci-lint
golangci-lint: $(GO_INSTALL_TOOL) ## Download golangci-lint locally if necessary.
	@$(GO_INSTALL_TOOL) \
		--target "$(GOLANGCI_LINT)" \
		--package github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		--tool-version "$(GOLANGCI_LINT_VERSION)"
