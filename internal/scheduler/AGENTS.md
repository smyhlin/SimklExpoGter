# AGENTS.md

## Purpose
OS-native scheduling abstraction and platform backends.

## Ownership
`internal/scheduler/`.

## Local Contracts
- Windows uses Task Scheduler.
- Linux uses systemd user timers and the wrapper script model.
- Do not introduce a daemon, tray process, or cron-only design.
- Keep platform stubs explicit.

## Work Guidance
- Preserve unit/task naming and schedule state fields.
- Keep Linux ExecStart quoting handled by the wrapper script.
- Keep stale-backup behavior and lingering support aligned with the service layer.

## Verification
- `go test ./internal/scheduler`
- On Linux scheduler changes, run the documented `schedule enable` and `systemctl --user` smoke checks

## Child DOX Index
- None
