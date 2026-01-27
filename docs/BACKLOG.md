# sysreboot - Feature Backlog

## Overview
This backlog contains potential features, enhancements, and technical improvements for the sysreboot enhanced reboot tool.

---

## Priority 1: Critical Enhancements

### ✅ P1.1: Cancellation Support - COMPLETED
**Status**: ✅ Completed in v0.2.0  
**See**: [DONE.md](DONE.md) for full implementation details

**Implemented Features**:
- `--cancel` flag to cancel pending scheduled actions
- `--status` flag to view pending scheduled actions
- Signal handling (Ctrl+C) for interactive cancellation
- PID file management for tracking scheduled actions
- Automatic cleanup of stale PID files
- Full test coverage with 19 new tests

---

### P1.2: Dry Run Mode
**Status**: Proposed  
**Effort**: Low  
**Impact**: High

Test commands without executing system actions.

**Features**:
- `--dry-run` or `--test` flag
- Show what would be executed without doing it
- Validate all flags and scheduling logic
- Display final command that would be run

**Example**:
```bash
sysreboot --reboot --delay 10 --dry-run
# Output:
# [DRY RUN] Would execute: systemctl reboot
# [DRY RUN] After delay: 10 minutes
# [DRY RUN] With message: (none)
```

**Technical Notes**:
- Add `dryRun` flag
- Wrap all system commands with dry-run check
- Useful for testing and automation scripts

---

### P1.3: Configuration File Support
**Status**: Proposed  
**Effort**: Medium  
**Impact**: Medium

Support configuration files for default settings.

**Features**:
- Load defaults from `/etc/sysreboot/config.yaml` or `~/.sysreboot.yaml`
- Override config with command-line flags
- Support for:
  - Default confirmation timeout
  - Default message templates
  - Preferred action (reboot vs poweroff)
  - Logging preferences

**Example Config** (`~/.sysreboot.yaml`):
```yaml
defaults:
  confirm: true
  confirm_timeout: 30
  verbose: true
  
messages:
  reboot: "System reboot initiated by ${USER}"
  poweroff: "System shutdown initiated by ${USER}"

logging:
  level: info
  file: /var/log/sysreboot.log
```

**Technical Notes**:
- Use `gopkg.in/yaml.v3` or `github.com/spf13/viper`
- Precedence: CLI flags > config file > defaults

---

## Priority 2: User Experience Improvements

### P2.1: Interactive Mode
**Status**: Proposed  
**Effort**: Medium  
**Impact**: Medium

Interactive menu when run without arguments.

**Features**:
- Present menu with options (reboot, poweroff, halt, schedule, cancel)
- Prompt for confirmation, delay, message
- Save preferences for future use
- Colorized output with `github.com/fatih/color`

**Example**:
```bash
$ sysreboot

sysreboot v0.2.0 - Enhanced System Reboot Tool
=============================================

What would you like to do?
  1) Reboot now
  2) Reboot after delay
  3) Reboot at specific time
  4) Power off now
  5) Power off after delay
  6) Cancel scheduled action
  7) Exit

Selection: _
```

**Technical Notes**:
- Use `github.com/manifoldco/promptui` or similar
- Save last used settings

---

### P2.2: Progress Indicators
**Status**: Proposed  
**Effort**: Low  
**Impact**: Low

Show countdown and progress during delays.

**Features**:
- Real-time countdown display for delays
- Progress bar for long delays (>5 minutes)
- Periodic reminders before action executes
- Option to suppress for scripting (`--quiet`)

**Example**:
```bash
$ sysreboot --reboot --delay 5
System will reboot in 5 minutes...
⏳ Time remaining: 04:58 (Press Ctrl+C to cancel)
```

**Technical Notes**:
- Update console output every second or minute
- Use ANSI escape codes for in-place updates
- Respect `--quiet` flag for automation

---

### P2.3: Better Error Messages
**Status**: Proposed  
**Effort**: Low  
**Impact**: Medium

Improve error messages with actionable suggestions.

**Features**:
- Detect common issues (no sudo, conflicting flags)
- Provide specific suggestions for resolution
- Show examples of correct usage
- Link to documentation

**Example**:
```bash
$ sysreboot --reboot
Error: Root privileges required

To run sysreboot, use one of the following:
  • sudo sysreboot --reboot
  • sudo !!

For more information, see: https://github.com/esobczak1970/sysreboot#usage
```

**Technical Notes**:
- Create error wrapper types
- Include suggestions in error messages

---

### P2.4: Multi-Language Support
**Status**: Proposed  
**Effort**: High  
**Impact**: Low

Internationalization (i18n) support.

**Features**:
- Translations for common languages (ES, FR, DE, JA, ZH)
- Detect system locale automatically
- `--lang` flag to override
- Localized messages and errors

**Technical Notes**:
- Use `golang.org/x/text` package
- Maintain translation files in `locales/` directory

---

## Priority 3: Advanced Features

### P3.1: Conditional Execution
**Status**: Proposed  
**Effort**: Medium  
**Impact**: Medium

Execute action only if certain conditions are met.

**Features**:
- `--if-idle-for <duration>` - only if system idle
- `--if-no-users` - only if no users logged in
- `--if-load-below <threshold>` - only if CPU load is low
- `--unless-process <name>` - skip if specific process running

**Example**:
```bash
# Reboot at 3 AM only if no users logged in
sysreboot --reboot --time 03:00 --if-no-users

# Shutdown if idle for 2 hours
sysreboot --poweroff --if-idle-for 2h
```

**Technical Notes**:
- Check system state before execution
- Use `w`, `uptime`, `ps` commands or `/proc` filesystem
- Log reason for skipping action

---

### P3.2: Pre/Post-Action Hooks
**Status**: Proposed  
**Effort**: Medium  
**Impact**: Medium

Execute custom scripts before/after system action.

**Features**:
- `--pre-hook <script>` - run before action
- `--post-hook <script>` - run after action (on reboot recovery)
- Save application state before reboot
- Verify successful restart

**Example**:
```bash
# Backup before reboot
sysreboot --reboot --pre-hook /usr/local/bin/backup.sh

# Run health check after reboot (via systemd)
sysreboot --reboot --post-hook /usr/local/bin/health-check.sh
```

**Technical Notes**:
- Execute hooks with timeout
- Handle hook failures (continue/abort)
- Post-hooks need systemd service or cron @reboot

---

### P3.3: Remote Execution
**Status**: Proposed  
**Effort**: High  
**Impact**: High

Reboot/shutdown remote systems via SSH.

**Features**:
- `--host <hostname>` for remote execution
- Support for SSH key authentication
- Parallel execution on multiple hosts
- Host groups from config file
- Status monitoring and reporting

**Example**:
```bash
# Single host
sysreboot --reboot --host server01.example.com

# Multiple hosts
sysreboot --reboot --hosts web01,web02,web03

# Host group from config
sysreboot --reboot --group webservers
```

**Technical Notes**:
- Use `golang.org/x/crypto/ssh` package
- Implement connection pooling
- Handle errors per host
- Consider using existing tools (Ansible, Fabric) via plugins

---

### P3.4: Scheduling Persistence
**Status**: Proposed  
**Effort**: Medium  
**Impact**: Medium

Scheduled actions survive system reboots.

**Features**:
- Store scheduled actions in database/file
- Restore pending schedules on boot
- Systemd service to manage schedules
- SQLite or JSON for storage

**Example**:
```bash
# Schedule persistent reboot
sysreboot --reboot --time 03:00 --persistent

# List persistent schedules
sysreboot --list-schedules

# Remove persistent schedule
sysreboot --remove-schedule <id>
```

**Technical Notes**:
- Create systemd service for schedule management
- Use `at` or `systemd-timers` as backend
- Maintain schedule database

---

### P3.5: Reboot Reason Tracking
**Status**: Proposed  
**Effort**: Low  
**Impact**: Low

Track and report reboot history with reasons.

**Features**:
- Record each reboot with timestamp and reason
- `--reason` flag to document why
- View reboot history with `--history`
- Export history to JSON/CSV

**Example**:
```bash
# Reboot with reason
sysreboot --reboot --reason "Kernel update"

# View history
sysreboot --history
# Output:
# 2026-01-27 03:00:00 | reboot  | Kernel update
# 2026-01-20 14:30:00 | reboot  | Security patches
# 2026-01-15 02:00:00 | poweroff| Maintenance window
```

**Technical Notes**:
- Store in `/var/log/sysreboot-history.log`
- Integrate with system logs
- Parse with `--history --format json`

---

## Priority 4: Technical Improvements

### P4.1: Better Testing Architecture
**Status**: Proposed  
**Effort**: High  
**Impact**: High

Refactor for 100% testability.

**Improvements**:
- Extract `main()` logic into testable functions
- Use dependency injection for system commands
- Mock interfaces for OS operations
- Integration tests with Docker containers
- CI/CD pipeline with automated testing

**Technical Notes**:
- Create `Commander` interface for exec operations
- Use table-driven tests extensively
- Add benchmark tests
- Test on multiple OS platforms (Linux, macOS, Windows)

---

### P4.2: Plugin System
**Status**: Proposed  
**Effort**: High  
**Impact**: Medium

Allow third-party extensions.

**Features**:
- Plugin API for custom actions
- Load plugins from directory
- Pre/post action hooks via plugins
- Plugin marketplace/registry

**Example Plugin**:
```go
// Plugin: slack-notify
type SlackNotifyPlugin struct {}

func (p *SlackNotifyPlugin) OnPreReboot(ctx context.Context) error {
    return sendSlackMessage("System rebooting...")
}
```

**Technical Notes**:
- Use `plugin` package or gRPC for plugins
- Define clear plugin API contract
- Sandbox plugin execution

---

### P4.3: Logging Enhancements
**Status**: Proposed  
**Effort**: Low  
**Impact**: Low

Better structured logging.

**Features**:
- JSON-structured logs option
- Log levels (DEBUG, INFO, WARN, ERROR)
- Syslog integration
- Log rotation
- Separate audit log

**Example**:
```bash
# Enable JSON logging
sysreboot --reboot --log-format json

# Set log level
sysreboot --reboot --log-level debug

# Send to syslog
sysreboot --reboot --syslog
```

**Technical Notes**:
- Use `github.com/sirupsen/logrus` or `go.uber.org/zap`
- Support multiple log outputs simultaneously

---

### P4.4: Performance Monitoring
**Status**: Proposed  
**Effort**: Medium  
**Impact**: Low

Track performance metrics.

**Features**:
- Measure time to execute actions
- Track success/failure rates
- Monitor system health before action
- Prometheus metrics endpoint

**Technical Notes**:
- Export metrics in Prometheus format
- Optional metrics collection (opt-in)

---

### P4.5: Security Hardening
**Status**: Proposed  
**Effort**: Medium  
**Impact**: High

Enhanced security features.

**Features**:
- Audit logging of all actions
- Role-based access control (RBAC)
- Require second factor authentication
- Signed binaries and updates
- Security policy enforcement

**Example**:
```bash
# Require 2FA for critical actions
sysreboot --poweroff --require-2fa

# Check security policy compliance
sysreboot --security-check
```

**Technical Notes**:
- Integrate with PAM for authentication
- Use SELinux/AppArmor policies
- Cryptographically sign releases

---

## Priority 5: Platform Expansion

### P5.1: FreeBSD Support
**Status**: Proposed  
**Effort**: Low  
**Impact**: Low

Add FreeBSD compatibility.

**Changes**:
- Support `shutdown` command syntax
- Test on FreeBSD systems
- Update documentation

---

### P5.2: Container Support
**Status**: Proposed  
**Effort**: Medium  
**Impact**: Medium

Graceful container shutdown/restart.

**Features**:
- Detect container environment (Docker, Kubernetes)
- Signal container orchestrator
- Graceful pod termination in Kubernetes
- Docker container restart

**Technical Notes**:
- Detect via environment variables
- Use Docker/K8s APIs when available

---

### P5.3: Systemd Integration
**Status**: Proposed  
**Effort**: Medium  
**Impact**: Medium

Deep systemd integration.

**Features**:
- Create systemd timer units
- Inhibitor locks for safe shutdown
- Journal integration
- Target dependencies

**Example**:
```bash
# Create systemd timer
sysreboot --reboot --time 03:00 --systemd-timer

# Check inhibitors
sysreboot --show-inhibitors
```

**Technical Notes**:
- Use D-Bus to communicate with systemd
- Implement `systemd-inhibit` style locking

---

## Priority 6: Documentation & Community

### P6.1: Enhanced Documentation
**Status**: Proposed  
**Effort**: Medium  
**Impact**: High

Comprehensive documentation.

**Deliverables**:
- Man pages (`man sysreboot`)
- Online documentation site
- Video tutorials
- Best practices guide
- Troubleshooting guide
- FAQ section

---

### P6.2: Package Distribution
**Status**: Proposed  
**Effort**: Medium  
**Impact**: High

Distribute via package managers.

**Targets**:
- Debian/Ubuntu: `.deb` package
- RHEL/CentOS: `.rpm` package
- Homebrew (macOS): `brew install sysreboot`
- Chocolatey (Windows): `choco install sysreboot`
- Snap package: `snap install sysreboot`
- AUR (Arch Linux)

---

### P6.3: Web Dashboard (Future)
**Status**: Proposed  
**Effort**: Very High  
**Impact**: Medium

Web-based management interface.

**Features**:
- Schedule reboots via web UI
- Monitor system status
- View reboot history
- Multi-host management
- REST API

**Technical Notes**:
- Separate project: `sysreboot-server`
- Built with Go backend + React frontend
- Authentication required

---

## Implementation Notes

### Development Workflow
1. Create feature branch from `main`
2. Implement feature with tests
3. Update documentation
4. Submit PR with changelog entry
5. Code review and merge

### Version Numbering
Follow semantic versioning:
- Major: Breaking changes
- Minor: New features (backward compatible)
- Patch: Bug fixes

### Contribution Guidelines
See `CONTRIBUTING.md` for:
- Code style guidelines
- Testing requirements
- PR process
- Release process

---

## Metrics for Success
- **Code Coverage**: Maintain >90% after refactoring
- **Response Time**: <100ms for command parsing
- **Error Rate**: <0.1% in production
- **User Satisfaction**: >4.5/5 stars
- **Documentation**: 100% of features documented

---

## Rejected Ideas

### Why Not Included

**GUI Application**: Out of scope - sysreboot is CLI-focused. Users wanting GUI should use system settings.

**Email Notifications**: Too specific - better handled by system monitoring tools or pre-hooks.

**Cloud Integration**: Too broad - consider separate tool or plugin.

**Database Backend**: Overengineered for current use case - flat files sufficient.

---

## Questions & Discussion

For feature requests, please:
1. Open an issue on GitHub
2. Describe use case
3. Propose implementation approach
4. Discuss with maintainers

**Repository**: https://github.com/esobczak1970/sysreboot  
**Discussions**: https://github.com/esobczak1970/sysreboot/discussions

---

*Last Updated: 2026-01-27*  
*Version: 0.2.0*  
*Completed Features: 1 (see DONE.md)*
