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

The default action is reboot. Destructive actions should be tested cautiously and with appropriate privileges.

Scheduling with `--time` or positive `--delay` **fails without
performing the system action** if its cancellation PID file cannot be created.
On Unix, `/var/run/sysreboot.pid` normally requires suitable privileges;
fix the permissions or run with the intended privilege rather than allowing
an uncancellable action. An immediate action does not need a PID file.

Cancellation no longer sends signals to the PID stored in a schedule file.
New scheduled actions record a random 128-bit registration identity. The
scheduler and `--cancel` compete for a single exclusive decision marker in
the trusted PID directory; only the first claim can succeed. The scheduler
checks for a decision every 250 ms and must claim it before beginning the
action. A successful cancellation request therefore prevents that registration
from starting execution, regardless of PID reuse, on Linux, macOS and Windows.

`--cancel` records cancellation rather than claiming the scheduler has exited;
`--status` distinguishes an undecided registration from a decision with
unconfirmed completion. Once execution has claimed the decision (including
the optional confirmation prompt), cancellation cannot be guaranteed. Ctrl+C
continues to stop a pending action in its original terminal.

**Upgrade and recovery:** Do not mix old and new versions of the CLI while a
scheduled action is pending. Finish or cancel any existing schedule in its
original terminal before upgrading all invocations. Legacy records without a
valid schedule identity are refused; stale records are intentionally preserved.
When recovering after a crash, first independently verify that the original
scheduler is no longer running and no OS shutdown has been dispatched. Only
then may an authorized operator remove its PID record and corresponding
`sysreboot-<schedule-id>.decision` marker from the trusted directory. Never
remove either during an active schedule. A missing or inaccessible decision
directory prevents scheduled execution. The local filesystem must support
exclusive creation; remote/shared PID directories are not supported.

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
