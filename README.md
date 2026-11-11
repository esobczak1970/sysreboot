# sysreboot

\`sysreboot\` is a Go command-line utility for rebooting, powering off, halting, scheduling, confirming, inspecting, and cancelling system actions with explicit safety controls.

Detailed usage notes are also available in [\`docs/README.md\`](docs/README.md).

## Requirements

- Go (version from `.go-version`) (required).
- Appropriate operating-system privileges for reboot, shutdown, or halt operations.
- Unix-specific features such as \`wall\` are only available on supported Unix-like systems.

## Quick start

\`\`\`bash
make check
make build
./bin/sysreboot --help
\`\`\`

Examples:

\`\`\`bash
./bin/sysreboot --reboot --delay 5 --message "System will reboot in 5 minutes"
./bin/sysreboot --poweroff --confirm
./bin/sysreboot --time "23:30"
./bin/sysreboot --status
./bin/sysreboot --cancel
\`\`\`

The default action is reboot. Destructive actions should be tested cautiously and with appropriate privileges. Cancellation validates the scheduled PID and sends an interrupt rather than forcibly killing the process if interrupt fails. An unsuccessful interrupt leaves the PID file in place for investigation; inspect stale schedules before retrying because operating systems may reuse process IDs.

Help and version commands run without opening the user log. Tests isolate user
logs and never execute real reboot or shutdown actions.

## Development

\`\`\`bash
make help
make doctor
make test
make test-race
make coverage
make lint
make modernize-check
make check
\`\`\`

`make modernize-check` invokes `go fix -diff ./...` with the pinned Go
version and reports modernization suggestions without changing the source.
Review any proposed changes before applying `go fix ./...`, then rerun the
canonical checks.

## Documentation

- [Feature and command overview](docs/README.md).
- [Backlog](docs/BACKLOG.md).
- [Completed work](docs/DONE.md).
- [Changelog](docs/CHANGELOG.md).
- [Historical coverage report](docs/TEST_COVERAGE_REPORT.md).

The repository \`LICENSE\` file contains the project license.

Go toolchain selection is centralized in `.go-version`. Module `go`/`toolchain` directives mirror that pin because Go requires literal versions. Run `make go-version-check` to verify the contract before building.
