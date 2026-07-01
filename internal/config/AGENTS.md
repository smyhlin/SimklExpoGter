# AGENTS.md

## Purpose
Persistent settings storage and defaults.

## Ownership
`internal/config/`.

## Local Contracts
- Own the settings.json schema, load/save behavior, and defaults.
- Keep config handling cross-platform.
- Preserve compatibility for existing JSON fields when practical.
- Keep access serialized through the store mutex.

## Work Guidance
- Update defaults carefully and keep derived values explicit.
- Keep user config paths and file permissions portable.
- Do not let higher layers reach around the store for persistence.

## Verification
- `go test ./internal/config`

## Child DOX Index
- None
