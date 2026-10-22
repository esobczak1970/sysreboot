# sysreboot

\`sysreboot\` is a Go command-line utility for rebooting, powering off, halting, scheduling, confirming, inspecting, and cancelling system actions with explicit safety controls.

Detailed usage notes are also available in [\`docs/README.md\`](docs/README.md).

## Requirements

- Go 1.27.3 (required).
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
make check
\`\`\`

## Documentation

- [Feature and command overview](docs/README.md).
- [Backlog](docs/BACKLOG.md).
- [Completed work](docs/DONE.md).
- [Changelog](docs/CHANGELOG.md).
- [Historical coverage report](docs/TEST_COVERAGE_REPORT.md).

The repository \`LICENSE\` file contains the project license.
