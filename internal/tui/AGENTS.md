# AGENTS.md

## Purpose
Dependency-free terminal user interface.

## Ownership
`internal/tui/`.

## Local Contracts
- Keep the TUI line-oriented and terminal-friendly.
- Do not add third-party UI dependencies.
- Preserve Windows VT handling and alternate-screen fallback behavior.
- Use the shared application service interface only.

## Work Guidance
- Keep prompts explicit and command-like.
- Keep SSH, tmux, and minimal shell usage smooth.
- Keep menu text aligned with CLI and docs.

## Verification
- `go test ./internal/tui`
- Manual `SimklExpoGter tui` smoke test when interaction changes

## Child DOX Index
- None
