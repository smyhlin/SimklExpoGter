# AGENTS.md

## Purpose
Export planning, flattening, and file writing.

## Ownership
`internal/exporter/`.

## Local Contracts
- Own CSV/JSON export shapes, filenames, and write behavior.
- Keep flattening logic consistent across GUI, CLI, and TUI.
- Do not mix network, auth, or scheduler concerns into exporter code.

## Work Guidance
- Preserve output compatibility unless a format change is intended.
- Add tests for row or filename changes.
- Keep file-system behavior explicit and deterministic.

## Verification
- `go test ./internal/exporter`

## Child DOX Index
- None
