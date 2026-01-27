# Changelog

All notable changes to sysreboot will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-01-27

### Added
- **Cancellation Support** - Cancel scheduled reboots/shutdowns
  - New `--cancel` / `-x` flag to cancel pending scheduled actions
  - New `--status` / `-st` flag to view pending scheduled actions
  - Interactive cancellation via Ctrl+C (SIGINT)
  - Signal handling for SIGTERM
  - PID file management for tracking scheduled actions
  - Automatic cleanup of stale PID files
  - Cross-platform PID file location handling

### Changed
- Updated version from 0.1.3 to 0.2.0
- Enhanced scheduling messages to indicate cancellation options
- Improved user feedback during delay periods

### Technical Details
- Added 19 new test functions for cancellation features
- Implemented `ScheduleInfo` struct for persistent schedule tracking
- Added signal handling with `os/signal` package
- JSON-based PID file format for structured data
- New functions: `getPIDFilePath()`, `writePIDFile()`, `readPIDFile()`, 
  `removePIDFile()`, `processExists()`, `cancelScheduledAction()`, 
  `showScheduleStatus()`

### Test Coverage
- All tests passing ✅
- Coverage: 54.8% of statements
- 19 new tests added
- Comprehensive edge case coverage

## [0.1.3] - 2026-01-27

### Changed
- Fixed resource leak in logger initialization
- Improved security with root privilege validation
- Enhanced error handling across all system commands
- Fixed confirmation timeout to cancel (safer default)
- Added context timeouts for all system commands
- Better error messages with detailed output

### Fixed
- Race condition in init() function
- Inconsistent confirmation logic
- Missing halt implementation for Linux
- Flag organization index mismatch

### Security
- Added privilege checking for Unix systems
- Removed sudo from internal commands (security vulnerability)
- Validates user privileges before execution

### Technical
- Added proper resource cleanup
- Implemented context timeouts
- Enhanced error propagation
- Improved test coverage from 0% to 56%

## [0.1.2] - 2023 (Original Release)

### Added
- Initial release by Eric Sobczak
- Basic reboot, poweroff, halt functionality
- Delay scheduling with `--delay`
- Time-based scheduling with `--time`
- Wall message broadcasting with `--message`
- Confirmation prompts with `--confirm`
- Verbose logging with `--verbose`
- Cross-platform support (Linux, macOS, Windows)

### Features
- Default action is reboot (prevents accidental shutdowns)
- Simple command-line interface
- MIT License

---

## Release Roadmap

### [0.3.0] - Planned
- Dry run mode (`--dry-run`)
- Configuration file support
- Interactive mode

### [0.4.0] - Planned
- Conditional execution (--if-idle, --if-no-users)
- Pre/post action hooks
- Enhanced logging

### [1.0.0] - Future
- Stable API
- Full feature completeness
- Production-ready
- Comprehensive documentation

---

[0.2.0]: https://github.com/esobczak1970/sysreboot/releases/tag/v0.2.0
[0.1.3]: https://github.com/esobczak1970/sysreboot/releases/tag/v0.1.3
[0.1.2]: https://github.com/esobczak1970/sysreboot/releases/tag/v0.1.2
