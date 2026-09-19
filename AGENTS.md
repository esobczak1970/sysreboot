# AGENTS.md

## Authority

This file is the canonical repository guidance for coding agents, including
Jules, Codex, Claude, and other automated contributors.

Follow direct user instructions first, then this file, then the maintained
project documentation. When documentation and the current repository disagree,
determine the intended current behavior and correct the contradiction in the
same change.

## Consistency and duplication

- Prefer one source of truth for every concept: configuration, schema, generated
  data, documentation, commands, and business rules.
- Treat duplicated implementation or setup logic as technical debt to eliminate,
  not a pattern to extend.
- Reuse established packages, helpers, scripts, and Make targets before creating
  another abstraction or workflow.
- Do not introduce parallel implementations merely to make a test, tool, or
  environment easier to satisfy.
- Fix root causes. Avoid compatibility layers, special cases, and copied schemas
  unless the project explicitly requires them.
- Generated files must be changed through their generator or canonical input,
  not hand-maintained independently.
- When consolidating duplicated code, preserve meaningful test coverage and
  behavior.

## Development workflow

- Use the root Makefile as the canonical command interface.
- Start with `make help` and `make doctor`.
- Run the smallest relevant test while developing and `make check` before
  finalizing a substantive change.
- Use `make ci` when reproducing the repository's canonical CI/static
  verification workflow.
- Do not add undocumented one-off build/test commands when an existing Make
  target can own the workflow.
- Keep dependency and tool versions aligned with the repository's committed
  configuration.

## Scope and code quality

- Make the smallest coherent root-cause change that fully solves the problem.
- Preserve existing behavior unless the task intentionally changes it.
- Prefer existing dependencies and standard-library/platform capabilities.
- Avoid broad mechanical rewrites unrelated to the requested change.
- Keep error handling explicit and actionable.
- Do not weaken tests, validation, constraints, or security checks simply to
  obtain a green build.
- Remove obsolete code and comments when their replacement is proven and the
  old path no longer serves a compatibility requirement.

## Tests

- Tests should validate the same behavior and architecture used in production
  whenever practical.
- Do not maintain a second implementation solely for tests.
- Test fixtures may simplify data, but they must not silently change core
  semantics.
- When infrastructure is required, repair or clearly report the infrastructure
  problem rather than substituting a materially different runtime.
- Add regression coverage for bug fixes and meaningful edge cases.

## Documentation

- Keep the root `README.md` accurate for current setup, build, test, and usage.
- Update maintained documentation in the same change when public behavior,
  architecture, commands, configuration, or source-of-truth rules change.
- Prefer links to canonical documentation over copying the same policy into
  multiple files.
- Keep Markdown structurally clean: one H1, ordered heading hierarchy, fenced
  code blocks with language identifiers, consistent list markers, and no
  unresolved merge-conflict markers.
- Historical plans and completed-work notes are context, not current
  implementation authority.

## Security and repository hygiene

- Never commit secrets, credentials, private keys, tokens, or local environment
  files.
- Preserve authorization, validation, escaping, and least-privilege behavior.
- Inspect the final diff for generated artifacts, temporary files, debugging
  output, and accidental unrelated changes.
- Do not leave disabled tests, placeholder implementations, or TODOs claiming
  work is complete when it is not.

## Project-specific guidance

**Project:** sysreboot

**Primary stack:** Go 1.27.1 system utility

- This utility can perform destructive system actions. Preserve explicit safety,
  confirmation, cancellation, privilege checks, and actionable errors.
- Keep platform-specific command execution centralized.
- Tests must never trigger real reboot/shutdown operations.
