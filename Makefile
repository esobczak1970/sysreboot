# Makefile
# Canonical local development and verification commands for sysreboot.

SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help
.DELETE_ON_ERROR:

APP_NAME := sysreboot
ENTRY := .
GO ?= go
GO_VERSION ?= $(shell tr -d '[:space:]' < .go-version)
BIN_DIR ?= $(CURDIR)/bin
COVERAGE_FILE ?= coverage.out
# golangci-lint v2 supports Go 1.27. Never downgrade to v1 as a workaround;
# rebuild the pinned v2 binary with the repository Go toolchain instead.
GOLANGCI_LINT_VERSION ?= v2.14.0
GOLANGCI_LINT_MODULE ?= github.com/golangci/golangci-lint/v2/cmd/golangci-lint
GOLANGCI_LINT_TIMEOUT ?= 5m
GOLANGCI_LINT_BIN ?= $(BIN_DIR)/golangci-lint
GO_TEST_FLAGS ?=
PKGS ?= ./...

.PHONY: help all doctor deps tidy update format format-check vet lint test test-race coverage build run check ci clean golangci-lint-bin

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
	@actual="$(GOTOOLCHAIN=local $(GO) env GOVERSION)"; expected="go$(GO_VERSION)"; \
	if [[ "$actual" != "$expected" ]]; then echo "Go toolchain mismatch: expected $expected from .go-version, got $actual." >&2; exit 1; fi
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

golangci-lint-bin: doctor
	@case "$(GOLANGCI_LINT_VERSION)" in \
	  v2.*) ;; \
	  *) echo "GOLANGCI_LINT_VERSION must remain on v2.x; do not downgrade to v1." >&2; exit 1 ;; \
	esac
	@mkdir -p "$(BIN_DIR)"
	@expected="$(GOLANGCI_LINT_VERSION)"; expected="$${expected#v}"; current_go="$$($(GO) env GOVERSION)"; rebuild=0; \
	if [[ ! -x "$(GOLANGCI_LINT_BIN)" ]]; then rebuild=1; else \
	  installed="$$($(GOLANGCI_LINT_BIN) --version | awk '{for (i=1;i<=NF;i++) if ($$i=="version") {print $$(i+1); exit}}')"; \
	  built_go="$$($(GO) version -m "$(GOLANGCI_LINT_BIN)" | awk 'NR==1 {print $$2}')"; \
	  [[ "$$installed" == "$$expected" && "$$built_go" == "$$current_go" ]] || rebuild=1; fi; \
	if [[ $$rebuild -eq 1 ]]; then echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) with $$current_go..."; GOBIN="$(BIN_DIR)" $(GO) install $(GOLANGCI_LINT_MODULE)@$(GOLANGCI_LINT_VERSION); fi; \
	installed="$$($(GOLANGCI_LINT_BIN) --version | awk '{for (i=1;i<=NF;i++) if ($$i=="version") {print $$(i+1); exit}}')"; \
	built_go="$$($(GO) version -m "$(GOLANGCI_LINT_BIN)" | awk 'NR==1 {print $$2}')"; \
	[[ "$$installed" == "$$expected" && "$$built_go" == "$$current_go" ]] || { echo "golangci-lint verification failed: expected $$expected built with $$current_go, got $$installed built with $$built_go." >&2; exit 1; }

lint: golangci-lint-bin
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
