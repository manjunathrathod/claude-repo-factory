# Development entry points for claude-repo-factory.
# Every target here is also what CI runs, so a green `make check` means a
# green pipeline.

BINARY      := claude-repo-factory
CMD         := ./cmd/claude-repo-factory
BIN_DIR     := bin
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE      := github.com/manjunathrathod/claude-repo-factory
LDFLAGS     := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.DEFAULT_GOAL := check

# The validation flow runs in a fixed order; never let make reorder or
# parallelise it.
.NOTPARALLEL:

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the binary into bin/
	@mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD)

.PHONY: install
install: ## Install the binary into GOBIN
	go install -trimpath -ldflags "$(LDFLAGS)" $(CMD)

.PHONY: test
test: ## Run the test suite
	go test ./...

.PHONY: race
race: ## Run the test suite under the race detector
	go test -race ./...

.PHONY: cover
cover: ## Run tests and report coverage per function
	go test -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -func=coverage.txt

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format all Go sources
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Fail if any source is not gofmt-formatted
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt-formatted:"; echo "$$unformatted"; exit 1; \
	fi

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	go mod tidy

.PHONY: vulncheck
vulncheck: ## Scan dependencies for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: check
check: ## The validation flow: gofmt -> go vet -> go test -> golangci-lint -> go build
	@bash scripts/check.sh

.PHONY: check-fast
check-fast: ## The validation flow without golangci-lint (the slow step)
	@bash scripts/check.sh --skip-lint

.PHONY: clean
clean: ## Remove build and coverage output
	rm -rf $(BIN_DIR) coverage.txt coverage.html
