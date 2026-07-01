# AGENTS.md

## Purpose
Simkl API client, auth, and data fetch helpers.

## Ownership
`internal/simkl/`.

## Local Contracts
- Own Simkl HTTP requests, device PIN flow, token exchange, and data fetch helpers.
- Keep request paths, headers, and query parameters explicit.
- Preserve the allowed types, statuses, and extended values used by the app.

## Work Guidance
- Keep API compatibility and error handling clear.
- Add tests when changing endpoints or request payloads.
- Keep auth behavior consistent with the CLI/TUI PIN flow.

## Verification
- `go test ./internal/simkl`

## Child DOX Index
- None
