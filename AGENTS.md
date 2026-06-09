# AGENTS.md

Repository instructions for AI coding agents and human maintainers.

## Project summary

SimklExpoGter is a Go/Wails/Svelte desktop + CLI + TUI app for exporting and backing up Simkl data.

Supported modes:

- Wails desktop GUI
- headless CLI
- dependency-free terminal UI
- Windows Task Scheduler recurring backups
- Linux systemd user timer recurring backups

The project is intentionally cross-platform. Do not reintroduce Windows-only assumptions.

## Ground rules

- Keep GUI, CLI and TUI behavior backed by the same service layer.
- Keep TUI dependency-free unless there is a strong reason to change that.
- Keep Linux scheduling based on systemd user timers.
- Keep Windows scheduling based on Task Scheduler.
- Keep CLI/TUI Simkl auth on PIN login by default.
- Do not use OAuth redirect URI login for CLI/TUI unless explicitly working on registered-redirect web/GUI behavior.
- Do not commit generated local build outputs.
- Do not commit secrets, tokens, local settings, or exported user backups.
- Prefer small, testable service-layer changes over UI-only fixes.

## Build commands

Linux:

```bash
./scripts/bootstrap.sh
./build.sh check
./build.sh linux
./build.sh linux-cli
```

Windows:

```bat
scripts\bootstrap.bat
build.bat check
build.bat windows
build.bat windows-cli
```

PowerShell:

```powershell
.\scripts\bootstrap.ps1
.\build.ps1 windows
```

## Validation checklist

Before committing, run what is available on your platform:

```bash
./build.sh check
./build.sh linux
./build.sh linux-cli
```

For Linux scheduler changes, also run:

```bash
./build/bin/SimklExpoGter schedule enable --frequency daily --time 02:00 --max-backup-age 24h --format both --field-mode all --content shows,movies,anime
systemctl --user status SimklExpoGterRecurringBackup.timer --no-pager --full
systemctl --user list-timers '*SimklExpoGter*'
```

For CLI/TUI auth changes, verify the PIN flow:

```bash
./build/bin/SimklExpoGter auth login
./build/bin/SimklExpoGter auth status
./build/bin/SimklExpoGter tui
```

## Important files

- `internal/appsvc/service.go` — shared application service layer
- `internal/cli/cli.go` — CLI command routing
- `internal/tui/tui.go` — dependency-free terminal UI
- `internal/scheduler/` — OS scheduler abstraction/backends
- `internal/simkl/client.go` — Simkl API and PIN auth client
- `internal/exporter/` — export writer/flattening
- `frontend/src/` — Svelte GUI
- `scripts/` — bootstrap scripts
- `build.sh`, `build.bat`, `build.ps1` — build entrypoints
- `docs/` — user/developer docs
- `packaging/` — Linux packaging assets

## Auth model

CLI/TUI should use Simkl PIN login:

```bash
SimklExpoGter auth login
```

The browser OAuth redirect flow is only for advanced apps with a redirect URI registered in the Simkl developer dashboard. Do not make `urn:ietf:wg:oauth:2.0:oob` the default again.

## Linux scheduler model

Linux recurring backups are installed as systemd user timer files under:

```text
~/.config/systemd/user/
```

The app writes:

```text
SimklExpoGterRecurringBackup.timer
SimklExpoGterRecurringBackup.service
SimklExpoGterRecurringBackup-run.sh
```

The service runs the wrapper script to avoid fragile systemd `ExecStart` quoting.

The app tries to enable user lingering automatically during schedule setup so timers can run after logout/boot.

## Release notes policy

For user-facing changes, update:

- `CHANGELOG.md`
- `` if release process changes
- `README.md` if setup/usage changes

## Style

- Keep docs direct and command-oriented.
- Prefer explicit examples over vague explanations.
- Keep terminal commands copy-paste friendly.
- Keep errors actionable.
