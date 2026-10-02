#!/usr/bin/env bash

set -Eeuo pipefail

# ---------------------------------------------------------------------------
# Go repository setup
#
# Standard behavior:
#   - Go 1.27.1 is the baseline toolchain.
#   - An older system/bootstrap Go (for example Go 1.24.3) is acceptable.
#   - Go's built-in toolchain manager downloads/uses Go 1.27.1 automatically.
#   - A repository requiring a newer Go version may automatically move forward.
#   - GOTOOLCHAIN=local is intentionally NOT used.
#
# Override when intentionally advancing the organization-wide baseline:
#
#   GO_VERSION=1.28.0 ./scripts/setup.sh
#
# ---------------------------------------------------------------------------

readonly DEFAULT_GO_VERSION="1.27.1"
readonly GO_VERSION="${GO_VERSION:-${DEFAULT_GO_VERSION}}"
readonly GO_TOOLCHAIN="go${GO_VERSION}+auto"

# Resolve the repository root robustly.
#
# Some coding-agent environments execute this script's contents through
# "bash -c" or source it from stdin. In those cases BASH_SOURCE may be unset,
# so the Git worktree is the authoritative location.
if command -v git >/dev/null 2>&1; then
    REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || true)"
else
    REPO_ROOT=""
fi

if [[ -z "${REPO_ROOT}" ]]; then
    if [[ -n "${BASH_SOURCE[0]:-}" ]]; then
        SCRIPT_DIR="$(
            cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
            pwd -P
        )"
        REPO_ROOT="$(
            cd -- "${SCRIPT_DIR}/.." >/dev/null 2>&1
            pwd -P
        )"
    else
        REPO_ROOT="$(pwd -P)"
    fi
fi

[[ -n "${REPO_ROOT}" ]] || {
    printf '[setup] ERROR: Unable to determine repository root.\n' >&2
    exit 1
}

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
    #   1.27.1
    #   go1.27.1
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

    # Persist the baseline in Go's user-level environment.
    #
    # The +auto suffix is important:
    #
    #   * go1.27.1 is the minimum/default toolchain.
    #   * repositories can require a newer toolchain later.
    #   * an older bundled/system Go does not become the effective build
    #     toolchain merely because it happens to be first in PATH.
    go env -w "GOTOOLCHAIN=${GO_TOOLCHAIN}"

    # Also export it for this setup process so there is no ambiguity if the
    # user's existing GOENV has not yet been re-read by a child process.
    export GOTOOLCHAIN="${GO_TOOLCHAIN}"
}

sync_go_version_file() {
    local version_file="${REPO_ROOT}/.go-version"

    if [[ ! -f "${version_file}" ]]; then
        log "Creating .go-version with ${GO_VERSION}"
        printf '%s\n' "${GO_VERSION}" >"${version_file}"
        return
    fi

    local configured
    configured="$(
        tr -d '[:space:]' <"${version_file}"
    )"
    configured="$(normalize_go_version "${configured}")"

    if [[ "${configured}" != "${GO_VERSION}" ]]; then
        log "Updating .go-version: ${configured} -> ${GO_VERSION}"
        printf '%s\n' "${GO_VERSION}" >"${version_file}"
    else
        log ".go-version already specifies ${GO_VERSION}"
    fi
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

check_for_forced_local_toolchain() {
    local found=0
    local candidates=(
        "${REPO_ROOT}/Makefile"
        "${REPO_ROOT}/makefile"
        "${REPO_ROOT}/GNUmakefile"
        "${REPO_ROOT}/scripts"
        "${REPO_ROOT}/.github"
    )

    local path

    for path in "${candidates[@]}"; do
        [[ -e "${path}" ]] || continue

        if grep -R \
            --exclude-dir=.git \
            --exclude='*.md' \
            --exclude='setup.sh' \
            -n \
            -E 'GOTOOLCHAIN[[:space:]]*=[[:space:]]*local|GOTOOLCHAIN=local' \
            "${path}" \
            >/dev/null 2>&1; then

            found=1

            warn "Repository contains GOTOOLCHAIN=local:"
            grep -R \
                --exclude-dir=.git \
                --exclude='*.md' \
                --exclude='setup.sh' \
                -n \
                -E 'GOTOOLCHAIN[[:space:]]*=[[:space:]]*local|GOTOOLCHAIN=local' \
                "${path}" \
                >&2 || true
        fi
    done

    if [[ "${found}" -eq 1 ]]; then
        cat >&2 <<EOF

[setup] WARNING:
GOTOOLCHAIN=local disables Go's automatic toolchain selection and can cause
builds to report an obsolete system/bootstrap Go version.

Replace those uses with either:

    GOTOOLCHAIN=${GO_TOOLCHAIN}

or preferably allow the environment configured by scripts/setup.sh to control
toolchain selection.

EOF
    fi
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

    local expected_prefix="go${GO_VERSION}"

    if [[ "${selected}" == "${expected_prefix}"* ]]; then
        return
    fi

    # With +auto, a newer toolchain is valid if go.mod/go.work requires it.
    #
    # We therefore verify that the requested baseline itself is available,
    # rather than incorrectly rejecting an intentional forward toolchain
    # selection.
    local baseline
    baseline="$(
        GOTOOLCHAIN="go${GO_VERSION}" go version 2>/dev/null |
            awk '{print $3}'
    )"

    if [[ "${baseline}" != "${expected_prefix}"* ]]; then
        die "Expected ${expected_prefix}, but Go selected ${baseline:-unknown}."
    fi

    log "Repository selected ${selected}; baseline ${expected_prefix} is installed."
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
    sync_go_version_file
    check_go_mod
    check_for_forced_local_toolchain
    download_toolchain
    verify_toolchain
    prepare_dependencies
    print_environment

    printf '\n'
    log "Setup complete."
    log "Effective toolchain: $(go version)"
}

main "$@"
