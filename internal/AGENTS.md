# AGENTS.md

## Purpose
Shared Go application implementation used by GUI, CLI, TUI, and platform integrations.

## Ownership
All packages under `internal/`.

## Local Contracts
- Keep GUI, CLI, and TUI behavior backed by the shared service layer.
- Keep CLI/TUI auth on PIN login by default.
- Keep scheduling OS-native: systemd user timers on Linux and Task Scheduler on Windows.
- Keep the TUI dependency-free.
- Keep package APIs small, explicit, and testable.

## Work Guidance
- Route cross-mode behavior through `internal/appsvc` instead of duplicating logic in UI or command layers.
- Preserve platform boundaries in scheduler and TUI code.
- Keep auth, export, config, and upload behavior consistent across modes.

## Verification
- `go test ./...`
- Target package tests for the package you changed

## Child DOX Index
- `internal/appsvc/AGENTS.md` — shared orchestration service
- `internal/cli/AGENTS.md` — headless command parser and runner
- `internal/config/AGENTS.md` — settings persistence and defaults
- `internal/exporter/AGENTS.md` — export planning and writers
- `internal/gdrive/AGENTS.md` — Google Drive auth and upload service
- `internal/scheduler/AGENTS.md` — OS scheduler abstraction and backends
- `internal/simkl/AGENTS.md` — Simkl API client and PIN auth
- `internal/tui/AGENTS.md` — dependency-free terminal UI
