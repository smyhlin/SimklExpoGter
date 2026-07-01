# AGENTS.md

## Purpose
Headless command parsing and command execution.

## Ownership
`internal/cli/`.

## Local Contracts
- CLI owns argument parsing, help text, and exit codes.
- Keep command behavior aligned with `internal/appsvc` and `internal/tui`.
- Preserve PIN login as the default auth flow for CLI and TUI use.
- Avoid GUI-only assumptions in CLI code.

## Work Guidance
- Keep flags and help output stable unless the user contract changes.
- Reuse shared service calls instead of reimplementing business rules.
- Keep the CLI usable on systems without a desktop stack.

## Verification
- `go test ./internal/cli`

## Child DOX Index
- None
