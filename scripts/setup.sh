#!/usr/bin/env bash

set -Eeuo pipefail

# .go-version is the only toolchain pin. Update it and the mirrored go.mod
# together; make go-version-check rejects drift. Setup never rewrites the pin.

# Resolve this script's repository, including stdin execution by setup runners.
if [[ -n "${BASH_SOURCE[0]:-}" && -f "${BASH_SOURCE[0]}" ]]; then
    REPO_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
elif command -v git >/dev/null 2>&1; then
    REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd -P)"
else
    REPO_ROOT="$(pwd -P)"
fi

pinned_go_version="$(tr -d '[:space:]' < "${REPO_ROOT}/.go-version")"
if [[ -n "${GO_VERSION:-}" && "${GO_VERSION}" != "${pinned_go_version}" ]]; then
    echo "GO_VERSION must match .go-version; edit the repository pin instead." >&2
    exit 1
fi
readonly GO_VERSION="${pinned_go_version}"
readonly GO_TOOLCHAIN="go${GO_VERSION}"
export GO_VERSION GO_TOOLCHAIN
bash "${REPO_ROOT}/scripts/check-go-version.sh"

log() {
    printf '[setup] %s\n' "$*"
}

warn() {
    printf '[setup] WARNING: %s\n' "$*" >&2
}

die() {
    printf '[setup] ERROR: %s\n' "$*" >&2
    exit 1
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

normalize_go_version() {
    local version="$1"

    # Accept either:
    #   the version read from .go-version
    # Toolchain version comes from .go-version.
    version="${version#go}"

    printf '%s\n' "${version}"
}

validate_go_version() {
    local version="$1"

    if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?([a-z]+[0-9]+)?$ ]]; then
        die "Invalid Go version: ${version}"
    fi
}

bootstrap_go_version() {
    GOTOOLCHAIN=local go version 2>/dev/null | awk '{print $3}' || true
}

selected_go_version() {
    go version 2>/dev/null | awk '{print $3}' || true
}

check_bootstrap_go() {
    if ! command_exists go; then
        cat >&2 <<'EOF'
[setup] ERROR: A bootstrap Go installation was not found in PATH.

Install any supported Go release with toolchain-management support, then rerun
this script. The bootstrap installation does not need to be the repository's
required Go version; Go will automatically obtain the required toolchain.

macOS with Homebrew:

    brew install go

EOF
        exit 1
    fi
}

configure_toolchain() {
    log "Configuring Go toolchain baseline: go${GO_VERSION}"

    # Persist the exact repository pin for shells launched after setup.
    # Deliberately omit +auto so module operations cannot select a newer Go.
    go env -w "GOTOOLCHAIN=${GO_TOOLCHAIN}"

    # Also export it for this setup process so there is no ambiguity if the
    # user's existing GOENV has not yet been re-read by a child process.
    export GOTOOLCHAIN="${GO_TOOLCHAIN}"
}

check_go_version_file() {
    bash "${REPO_ROOT}/scripts/check-go-version.sh"
}

check_go_mod() {
    local go_mod="${REPO_ROOT}/go.mod"

    if [[ ! -f "${go_mod}" ]]; then
        warn "No go.mod found at repository root."
        return
    fi

    local module_go_version
    local module_toolchain

    module_go_version="$(
        awk '$1 == "go" { print $2; exit }' "${go_mod}"
    )"

    module_toolchain="$(
        awk '$1 == "toolchain" { print $2; exit }' "${go_mod}"
    )"

    if [[ -n "${module_go_version}" ]]; then
        log "go.mod language version: ${module_go_version}"
    fi

    if [[ -n "${module_toolchain}" ]]; then
        log "go.mod preferred toolchain: ${module_toolchain}"
    fi

    # We deliberately do NOT rewrite the `go` directive here.
    #
    # `go 1.x.y` is part of the module's compatibility contract and should be
    # changed intentionally rather than as a side effect of environment setup.
    #
    # The effective development/build toolchain is controlled by GOTOOLCHAIN.
}

download_toolchain() {
    log "Ensuring go${GO_VERSION} is available..."

    # Explicitly request the baseline. If it is not already cached, the Go
    # command downloads and verifies it through the standard toolchain
    # mechanism.
    GOTOOLCHAIN="go${GO_VERSION}" go version >/dev/null

    log "go${GO_VERSION} is available."
}

verify_toolchain() {
    local selected
    selected="$(selected_go_version)"

    if [[ -z "${selected}" ]]; then
        die "Unable to determine selected Go toolchain."
    fi

    log "Effective Go toolchain: ${selected}"

    local expected="go${GO_VERSION}"
    [[ "${selected}" == "${expected}" ]] || die "Expected ${expected}, but Go selected ${selected}."

}

prepare_dependencies() {
    local go_mod="${REPO_ROOT}/go.mod"

    if [[ ! -f "${go_mod}" ]]; then
        return
    fi

    log "Downloading Go module dependencies..."
    (
        cd "${REPO_ROOT}"
        go mod download
    )
}

print_environment() {
    log "Repository: ${REPO_ROOT}"
    log "GOTOOLCHAIN: $(go env GOTOOLCHAIN)"
    log "GOOS:        $(go env GOOS)"
    log "GOARCH:      $(go env GOARCH)"
    log "GOPATH:      $(go env GOPATH)"
    log "GOMODCACHE:  $(go env GOMODCACHE)"
}

main() {
    validate_go_version "${GO_VERSION}"

    log "Setting up Go development environment"
    log "Required baseline: go${GO_VERSION}"

    check_bootstrap_go

    local bootstrap
    bootstrap="$(bootstrap_go_version)"

    if [[ -n "${bootstrap}" ]]; then
        log "Bootstrap Go command: ${bootstrap}"
    fi

    configure_toolchain
    check_go_version_file
    check_go_mod
    download_toolchain
    verify_toolchain
    prepare_dependencies
    print_environment

    printf '\n'
    log "Setup complete."
    log "Effective toolchain: $(go version)"
}

main "$@"
