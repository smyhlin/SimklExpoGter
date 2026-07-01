# AGENTS.md

## Purpose
Shared application orchestration for config, auth, export, scheduling, and backup state.

## Ownership
`internal/appsvc/`.

## Local Contracts
- This package owns the shared service layer used by GUI, CLI, and TUI.
- Keep orchestration mode-agnostic and deterministic.
- Centralize validation, defaults, and summary structures here.
- Do not move transport- or UI-specific behavior into this package.

## Work Guidance
- Add service methods instead of duplicating orchestration in callers.
- Keep result structs stable and explicit.
- Keep tests focused on shared behavior.

## Verification
- `go test ./internal/appsvc`

## Child DOX Index
- None
