# SimklExpoGter

`SimklExpoGter` is a single-binary Simkl backup/export application built with Go, Wails v2, Svelte 5 and TypeScript.

It supports four practical modes from the same codebase:

- desktop GUI for normal interactive use
- TUI for guiless terminals, SSH sessions and fallback when Linux has no GUI session
- headless CLI for scripts and scheduled jobs
- recurring background backups through Windows Task Scheduler or Linux systemd user timers

GUI, TUI and CLI modes share the same settings file, export directory, auth token and export engine.

## Current Abilities

- Save Simkl `client_id`, `client_secret` and export directory
- Authorize with Simkl through device login in the GUI, Simkl PIN login in the GUI, CLI and TUI; OAuth code exchange is kept only for web apps with a registered redirect URI
- Run a one-click full backup from the `Easy Export` tab
- Run selective exports from the `Advanced Export` tab or CLI with media/status filters
- Export to `csv`, `json` or `both`
- Use `all` or `compact` field modes
- Use `single-file` or `separate-files` grouping
- Include episode files, memos, next-watch info and watched-at timestamps
- Use activity-aware incremental exports through `/sync/activities`
- Upload generated backups to Google Drive
- Configure recurring daily or weekly backups
- Build for Windows and Linux with included `.bat`, `.ps1` and `.sh` scripts

## Supported Export Shapes

- Media types: `shows`, `movies`, `anime`
- Statuses: `watching`, `plantowatch`, `hold`, `completed`, `dropped`
- Extended modes: `full`, `full_anime_seasons`, `simkl_ids_only`, `ids_only`
- Output formats: `csv`, `json`, `both`
- Field modes: `all`, `compact`
- Grouping: `single-file`, `separate-files`

## Requirements

Build requirements:

- Go `1.25+`, matching `go.mod`
- Node.js and npm
- Wails CLI `v2.11.0+`
- Linux GUI builds need GCC/build tools, `pkg-config`, GTK3 and WebKitGTK
- Windows GUI builds need Microsoft WebView2 runtime, normally already installed on Windows 10/11

Runtime requirements:

- GUI mode requires a working desktop/WebView environment
- TUI and CLI modes do not require Node.js, npm or a desktop session
- Linux scheduling requires `systemctl --user` and a working systemd user session
- Windows scheduling requires Task Scheduler

## Bootstrap

Bootstrap installs or checks build prerequisites, installs the Wails CLI, downloads Go modules and installs frontend dependencies.

Linux, Arch, EndeavourOS, Debian, Ubuntu:

```bash
./scripts/bootstrap.sh
```

Useful Linux bootstrap flags:

```bash
./scripts/bootstrap.sh --no-system-deps
./scripts/bootstrap.sh --no-wails
./scripts/bootstrap.sh --no-frontend
./scripts/bootstrap.sh --no-tidy
./scripts/bootstrap.sh --system-upgrade
```

On Arch/EndeavourOS, bootstrap installs only missing dependency packages by default and does **not** run a full system upgrade. Use `--system-upgrade` only when you intentionally want bootstrap to run `pacman -Syu` before installing dependencies.

Windows:

```bat
scripts\bootstrap.bat
```

Windows bootstrap uses PowerShell internally and can use `winget` to install missing Go, Node.js LTS and Git when available. If `winget` is not available, install those manually and rerun bootstrap.

## Build And Test

Linux commands:

```bash
./build.sh check
./build.sh linux
./build.sh linux-cli
./build.sh deb
./build.sh appimage
./build.sh arch
./build.sh release-linux
```

Windows commands:

```bat
build.bat check
build.bat windows
build.bat windows-cli
build.bat release-windows
```

PowerShell wrapper:

```powershell
.\build.ps1 windows
.\build.ps1 windows-cli
```

Generated output locations:

- GUI Linux binary: `dist/linux/SimklExpoGter`
- CLI/TUI Linux binary: `dist/linux/SimklExpoGter-cli`
- GUI Windows binary: `dist/windows/SimklExpoGter.exe`
- CLI/TUI Windows binary: `dist/windows/SimklExpoGter-cli.exe`
- Debian package: `dist/SimklExpoGter_<version>_amd64.deb`
- AppDir/AppImage work area: `dist/appimage/SimklExpoGter.AppDir`
- Arch package or package tree: `dist/SimklExpoGter_<version>-1-x86_64.pkg.tar.zst` or `dist/arch/pkg`

Clean generated outputs:

```bash
./build.sh clean
```

```bat
build.bat clean
```


## Release Builds

Linux release assets:

```bash
VERSION=0.1.0 ./build.sh release-linux
```

This builds:

- Linux GUI tarball
- Linux CLI/TUI tarball
- Windows CLI/TUI zip through Go cross-compilation
- `.deb` package when `dpkg-deb` is available
- Arch package when `bsdtar` is available
- `SHA256SUMS.txt`

Linux GUI builds auto-detect WebKitGTK through `pkg-config`. If only WebKitGTK 4.1 is available, the build uses the Wails `webkit2_41` tag automatically.

Windows release assets should be built on Windows:

```bat
set VERSION=0.1.0
build.bat release-windows
```

For the first GitHub release, prefer GitHub Release assets over GitHub Packages.

## Linux Notes

The Linux GUI build is Wails/WebKitGTK based. The bootstrap script detects `apt` or `pacman` and installs the closest matching GTK3/WebKitGTK development stack.

On Debian/Ubuntu, newer distributions may provide WebKitGTK 4.1 instead of 4.0. Bootstrap writes `.build/webkit.env`; `build.sh` reads it and adds the `webkit2_41` build tag when needed.

AppImage packaging is implemented as an AppDir/linuxdeploy flow:

1. `./build.sh appimage` builds the Linux GUI binary.
2. It prepares `dist/appimage/SimklExpoGter.AppDir`.
3. If `linuxdeploy` or `linuxdeploy-x86_64.AppImage` exists, it attempts to generate an AppImage.
4. If linuxdeploy is missing, the AppDir remains ready and the script prints the next command to run.

Because Wails v2 depends on system WebKitGTK, native `.deb`/Arch packages are still the cleanest Linux distribution format. AppImage is provided as a convenience target, not as the only Linux path.


### Linux recurring backup notes

Linux recurring backups are installed as a systemd user timer plus service and a small wrapper script under `~/.config/systemd/user/`. `schedule enable` tries to enable Linux user lingering automatically so timers can run after logout/boot. The wrapper script avoids fragile `ExecStart` argument quoting. Setup starts the timer in a non-persistent mode first, then rewrites it with `Persistent=true`, so configuring the schedule does not immediately launch a surprise catch-up job while future missed runs are still caught.

Troubleshooting commands:

```bash
systemctl --user status SimklExpoGterRecurringBackup.timer --no-pager --full
systemctl --user status SimklExpoGterRecurringBackup.service --no-pager --full
journalctl --user -u SimklExpoGterRecurringBackup.service -n 80 --no-pager
```

## Simkl Authentication

For desktop/terminal use, prefer the Simkl PIN flow:

```bash
./build/bin/SimklExpoGter config set --client-id YOUR_CLIENT_ID --output "$HOME/SimklExpoGterBackups"
./build/bin/SimklExpoGter auth login
```

Open the printed `https://simkl.com/pin/` URL, enter the displayed code, approve the app, and wait for the CLI to save the access token.

Why not `auth url`? Simkl's browser OAuth flow requires `redirect_uri` to match a URL registered in the Simkl developer dashboard byte-for-byte. For CLI/TUI/desktop binaries, the PIN flow avoids `redirect_uri` and does not require `client_secret`. The old `auth url` command is kept as a compatibility alias for PIN login. `auth oauth-url` and `auth exchange --code` are only for apps that have a real registered redirect URI.

## GUI Usage

Run the app with no subcommand:

```bash
SimklExpoGter
```

On Windows:

```powershell
SimklExpoGter.exe
```

Force GUI mode on Linux even if no `DISPLAY` or `WAYLAND_DISPLAY` is detected:

```bash
SimklExpoGter --gui
```

The GUI has three main work areas:

1. `Settings`, save credentials, manage auth, configure backup storage and recurring backups
2. `Easy Export`, run a full backup quickly
3. `Advanced Export`, run filtered and format-specific exports

Typical setup flow:

1. Create a Simkl app and copy the `client_id`
2. Open `SimklExpoGter`
3. Save the `client_id`
4. Save the `client_secret` if you want OAuth code exchange support
5. Choose an export directory or configure Google Drive
6. Complete auth
7. Run a backup
8. Optionally configure recurring backups

## TUI Usage

Run the self-contained terminal UI:

```bash
SimklExpoGter tui
```

Aliases:

```bash
SimklExpoGter --tui
SimklExpoGter terminal
```

Linux GUI fallback behavior:

- If no CLI command is provided and no `DISPLAY`/`WAYLAND_DISPLAY` exists, the app starts TUI automatically.
- If GUI startup fails and the process is attached to a real terminal, the app falls back to TUI.
- Use `--gui` to force GUI mode and disable fallback.

The TUI opens in a full-screen terminal workspace using the terminal alternate screen. It redraws in place instead of dumping a new menu after every command.

TUI keys:

- Arrow keys: move through the menu
- Enter: select item or submit form
- Tab: cycle settings fields
- Esc: return to main menu
- `q` or `Ctrl+C`: quit

Current TUI abilities:

- edit Simkl client ID, optional secret and export directory
- authenticate with Simkl PIN login
- clear saved access token
- run an easy export with saved/default options
- inspect and configure recurring backups
- show export result files and warnings

## Headless CLI Usage

CLI mode is activated by a recognized subcommand.

Root help:

```bash
SimklExpoGter help
SimklExpoGter --help
```

Current top-level commands:

- `help`
- `run`
- `config path`
- `config show`
- `config set`
- `auth login`
- `auth exchange`
- `auth status`
- `auth clear`
- `tui`

### Config Commands

Show the config file path:

```bash
SimklExpoGter config path
```

Show the current config summary as compact JSON:

```bash
SimklExpoGter config show
```

Persist config values:

```bash
SimklExpoGter config set --client-id "your-client-id" --secret "your-client-secret" --output "./exports"
```

Notes:

- `--secret` is an alias for `--client-secret`
- `config set` supports partial updates
- `--output` persists the default export directory for GUI, TUI and CLI use

### Auth Commands

Start Simkl PIN login:

```bash
SimklExpoGter auth login
```

Show auth readiness as compact JSON:

```bash
SimklExpoGter auth status
```

Legacy OAuth exchange for apps with a registered redirect URI:

```bash
SimklExpoGter auth exchange --code "PASTE_CODE_HERE"
```

Clear the saved access token:

```bash
SimklExpoGter auth clear
```

### Run Command

Show run help:

```bash
SimklExpoGter help run
```

Full backup-style export:

```bash
SimklExpoGter run --mode all --content anime,movie,series --output ./exports
```

Filtered compact export:

```bash
SimklExpoGter run --mode compact --status watching,completed --format both --grouping single-file --activity-check
```

Incremental export:

```bash
SimklExpoGter run --content anime --date-from "2026-01-01T00:00:00Z" --format json --filename-prefix anime-backup
```

Supported `run` flags:

- `--field-mode`, `--mode`: `all|compact`, default `all`
- `--types`, `--content`: comma-separated media types
- accepted media aliases: `movie|movies`, `series|show|shows`, `anime`
- `--status`: comma-separated statuses
- `--extended`: `full|full_anime_seasons|simkl_ids_only|ids_only`
- `--format`: `csv|json|both`, default `csv`
- `--grouping`: `single-file|separate-files`, default `separate-files`
- `--output`: export directory override for this run only
- `--filename-prefix`: default `simkl-export`
- `--date-from`: explicit incremental timestamp
- `--episode-files`: default `true`
- `--memos`: default `true`
- `--next-watch-info`: default `true`
- `--episode-watched-at`: default `true`
- `--activity-check`: default `false`

Exit codes:

- `0`: success
- `2`: usage or validation error
- `3`: missing config or auth prerequisite
- `4`: runtime, auth or export failure

## Recurring Backup

The app can create a recurring background backup from the `Settings` tab.

Supported schedulers:

- Windows: Task Scheduler
- Linux: systemd user timer
- macOS: not implemented yet

What recurring backups support:

- daily schedule
- weekly schedule with weekday selection
- configurable run time
- output format: `csv`, `json`, or `both`
- field mode: `all` or `compact`
- selected media content types
- optional activity check before export
- local export or Google Drive upload based on saved backup storage settings

How it behaves:

- the GUI does not need to stay open
- there is no tray process or always-on daemon
- the scheduler launches the same executable in headless `run` mode
- moving or renaming the configured executable after enabling the schedule can break the scheduler entry

Linux systemd user timer files are written under:

```text
~/.config/systemd/user/
```

The fixed task/unit base name is:

```text
SimklExpoGterRecurringBackup
```

## Config Storage

The shared settings file is stored in the OS user config directory.

Typical paths:

- Windows: `%APPDATA%\SimklExpoGter\settings.json`
- macOS: `~/Library/Application Support/SimklExpoGter/settings.json`
- Linux: `${XDG_CONFIG_HOME:-~/.config}/SimklExpoGter/settings.json`

Saved values include:

- `clientId`
- `clientSecret`
- `accessToken`
- `exportDirectory`
- `backup`
- `lastActivities`
- `schedule`

Default export directory fallback:

- Windows: `%USERPROFILE%\Downloads\SimklExpoGter`
- macOS/Linux: `~/Downloads/SimklExpoGter`

Persistence rules:

- GUI, TUI and CLI share the same config file
- `config set --output` persists the default export directory
- `run --output` overrides the output directory for one run only
- activity-aware exports update the saved `lastActivities` snapshot after a successful run
- recurring backup settings are stored in the shared config file
- next run, last run and last result metadata are owned by the OS scheduler

## Export Notes

- Activity-aware export checks `/sync/activities` before fetching export data
- If no prior activity snapshot exists, the first activity-aware export falls back to a full snapshot
- CSV exports flatten the Simkl payload into rows
- JSON exports preserve grouped payload structure
- Google Drive backups first write to a temporary local directory, upload generated files, then remove the temporary directory

## Repo Layout

- `main.go`: GUI entrypoint, CLI dispatch and Linux TUI fallback
- `main_cli.go`: CLI/TUI-only entrypoint used by `go build -tags cli`
- `app.go`: Wails adapter and GUI-only session state
- `internal/appsvc`: shared config, auth, scheduling and export orchestration
- `internal/cli`: headless CLI parser and command runner
- `internal/tui`: dependency-free line-oriented terminal interface
- `internal/config`: settings persistence
- `internal/scheduler`: Windows Task Scheduler, Linux systemd user timers and platform stubs
- `internal/simkl`: Simkl API client
- `internal/exporter`: export planning and writers
- `internal/gdrive`: Google Drive OAuth and upload service
- `frontend`: Svelte 5 + TypeScript desktop UI
- `scripts`: bootstrap helpers
- `packaging`: Linux desktop, AppImage, Debian and Arch packaging files

## Current Limits

- macOS scheduling is not implemented yet
- AppImage support is best-effort because Wails v2 GUI binaries depend on the system WebKitGTK runtime
- `go.sum` is refreshed by `go mod tidy` during bootstrap after dependency changes

## API Reference

The export behavior is modeled against the local `live Simkl API docs` reference, especially:

- `/oauth/pin`
- `/sync/activities`
- `/sync/all-items`

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).


## Linux autorun and stale scheduled backups

Linux recurring backups use systemd user timers. The generated timer uses `Persistent=true`, so missed runs can catch up after the machine/user manager becomes available again.

For unattended timers after logout or boot, enable lingering:

```bash
SimklExpoGter schedule linger
```

Scheduled runs use a stale-backup guard by default. The app stores `lastSuccessfulBackupAt` only after a successful export/upload. When the scheduler runs, `SimklExpoGter run --scheduled` checks whether the last successful backup is older than the configured threshold.

Example:

```bash
SimklExpoGter schedule enable --frequency daily --time 02:00 --max-backup-age 24h
```

If the PC was off and the last backup is older than `24h`, the next scheduled run creates a new backup. If the last backup is still fresh, it exits cleanly without duplicate output.
