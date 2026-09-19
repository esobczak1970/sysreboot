# sysreboot command guide

\`sysreboot\` is a privileged system-action utility for rebooting, powering off,
halting, scheduling, inspecting, and cancelling actions.

The root [README](../README.md) contains the project overview and development
workflow. This document focuses on runtime behavior.

## Safety model

The current implementation has these important defaults:

- The default action is an immediate reboot.
- The default delay is zero minutes.
- Confirmation is optional and is enabled with \`--confirm\` or \`-c\`.
- Scheduled or delayed actions write a PID/status file so they can be inspected
  or cancelled.
- On Unix-like systems the program requires root privileges before performing
  system actions.
- Confirmation expires after 10 seconds by default and cancels the action when
  no affirmative response is received.

Because the default action is destructive, use \`--help\`, \`--status\`, and
explicit action flags carefully.

## Build

\`\`\`bash
make check
make build
\`\`\`

The resulting executable is \`bin/sysreboot\`.

## Common actions

Immediate reboot:

\`\`\`bash
sudo ./bin/sysreboot --reboot
\`\`\`

Power off with confirmation:

\`\`\`bash
sudo ./bin/sysreboot --poweroff --confirm
\`\`\`

\`--shutdown\` is an alias for power off.

Halt:

\`\`\`bash
sudo ./bin/sysreboot --halt
\`\`\`

Delay an action by five minutes:

\`\`\`bash
sudo ./bin/sysreboot --reboot --delay 5 \
  --message "System will reboot in 5 minutes"
\`\`\`

Schedule an action for a local 24-hour clock time:

\`\`\`bash
sudo ./bin/sysreboot --reboot --time "23:30"
\`\`\`

## Scheduled-action management

Show the current scheduled action:

\`\`\`bash
sudo ./bin/sysreboot --status
\`\`\`

Cancel it:

\`\`\`bash
sudo ./bin/sysreboot --cancel
\`\`\`

Scheduled and delayed actions use a PID/status file. Unix systems prefer
\`/var/run/sysreboot.pid\` and fall back to a temporary location when needed.

## Messaging and verbosity

Broadcast a message before the action:

\`\`\`bash
sudo ./bin/sysreboot --reboot --message "Maintenance reboot"
\`\`\`

On Linux and macOS this uses the system \`wall\` command when available.

Enable verbose logging with:

\`\`\`bash
sudo ./bin/sysreboot --verbose
\`\`\`

## Platform behavior

- Linux uses \`systemctl\`.
- macOS uses \`shutdown\` or \`halt\` and requires root privileges.
- Windows uses the native \`shutdown\` command for reboot and power-off actions.

Exact operating-system behavior should be verified before using the utility on
production systems.

## Development

Use the root Makefile:

\`\`\`bash
make help
make doctor
make test
make test-race
make coverage
make lint
make check
\`\`\`

The command implementation in \`main.go\` is authoritative when older planning
or historical notes disagree with this guide.
