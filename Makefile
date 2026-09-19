# Makefile
# Canonical local development and verification commands for sysreboot.

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help
.DELETE_ON_ERROR:

APP_NAME := sysreboot
ENTRY := .
GO ?= go
BIN_DIR ?= $(CURDIR)/bin
COVERAGE_FILE ?= coverage.out
GOLANGCI_LINT_VERSION ?= v2.13.2
GOLANGCI_LINT_TIMEOUT ?= 5m
GOLANGCI_LINT_BIN ?= $(BIN_DIR)/golangci-lint
GO_TEST_FLAGS ?=
PKGS ?= ./...

.PHONY: help all doctor deps tidy update format format-check vet lint test test-race coverage build run check ci clean

help:
	@printf '%s\n' \
	  'sysreboot Make targets:' \
	  '  make doctor       - Verify the local Go development toolchain.' \
	  '  make deps         - Download module dependencies.' \
	  '  make format       - Apply gofmt simplifications to Go source.' \
	  '  make format-check - Fail when Go source is not gofmt clean.' \
	  '  make lint         - Run golangci-lint using the repository-pinned version.' \
	  '  make test         - Run the Go test suite.' \
	  '  make test-race    - Run tests with the race detector.' \
	  '  make coverage     - Write $(COVERAGE_FILE) with package coverage.' \
	  '  make build        - Compile the project.' \
	  '  make run          - Run sysreboot locally (ARGS="...").' \
	  '  make check        - Run formatting, vet, lint, tests, and build.' \
	  '  make ci           - Download dependencies and run make check.' \
	  '  make clean        - Remove generated build/test artifacts.' \
	  ''

all: build

doctor:
	@command -v "$(GO)" >/dev/null || { echo "Go is required but was not found in PATH." >&2; exit 1; }
	@$(GO) version
	@$(GO) env GOMOD

deps: doctor
	$(GO) mod download

tidy: doctor
	$(GO) mod tidy

update: doctor
	$(GO) get -u ./...
	$(GO) mod tidy

format: doctor
	@files="$$(find . -type f -name '*.go' -not -path './vendor/*' -not -path './bin/*')"; \
	if [[ -n "$$files" ]]; then gofmt -s -w $$files; fi

format-check: doctor
	@files="$$(find . -type f -name '*.go' -not -path './vendor/*' -not -path './bin/*')"; \
	bad="$$(if [[ -n "$$files" ]]; then gofmt -s -l $$files; fi)"; \
	if [[ -n "$$bad" ]]; then echo "The following Go files need formatting:"; echo "$$bad"; exit 1; fi

vet: deps
	$(GO) vet $(PKGS)

$(GOLANGCI_LINT_BIN): go.mod
	@mkdir -p "$(BIN_DIR)"
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	GOBIN="$(BIN_DIR)" $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: $(GOLANGCI_LINT_BIN)
	"$(GOLANGCI_LINT_BIN)" run --timeout="$(GOLANGCI_LINT_TIMEOUT)" $(PKGS)

test: deps
	$(GO) test -count=1 $(GO_TEST_FLAGS) $(PKGS)

test-race: deps
	$(GO) test -race -count=1 $(GO_TEST_FLAGS) $(PKGS)

coverage: deps
	$(GO) test -count=1 -coverprofile="$(COVERAGE_FILE)" $(GO_TEST_FLAGS) $(PKGS)
	$(GO) tool cover -func="$(COVERAGE_FILE)"

build: deps
	@mkdir -p "$(BIN_DIR)"
	$(GO) build -trimpath -o "$(BIN_DIR)/$(APP_NAME)" $(ENTRY)

run: deps
	$(GO) run $(ENTRY) $(ARGS)

check: format-check vet lint test build

ci: deps check

clean:
	rm -rf "$(BIN_DIR)" "$(COVERAGE_FILE)"
