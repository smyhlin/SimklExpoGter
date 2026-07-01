# Changelog

All notable changes to this project will be documented in this file.

## [0.1.0] - First public release

### Added

- Added Windows GUI release asset generation to the Linux release build flow.
- Cross-platform app structure with GUI, CLI and TUI modes.
- Linux GUI build support through Wails.
- Linux CLI/TUI-only build support with `go build -tags cli`.
- Windows CLI/TUI-only build support.
- Automatic Linux fallback to TUI when no GUI display is available.
- Dependency-free full-screen terminal dashboard for SSH/tmux/headless usage.
- Simkl PIN login for CLI/TUI/headless authentication.
- Legacy OAuth commands kept for advanced registered redirect URI use.
- Local CSV/JSON backup export.
- `csv`, `json` and `both` output modes.
- `all` and `compact` field modes.
- Shows/movies/anime content filters.
- Episode files, memos, next-watch info and episode watched-at timestamps.
- Activity-aware incremental export support.
- Google Drive backup upload support.
- Telegram bot backup upload support.
- Encrypted settings backup import/export.
- Windows Task Scheduler recurring backup support.
- Linux systemd user timer recurring backup support.
- Linux automatic user lingering setup for scheduled backups after logout/boot.
- Stale-backup guard for scheduled runs with `12h`, `24h`, `3d`, `1w` thresholds.
- Scheduler status reporting for installed state, next run, last run and last result.
- Linux timer wrapper script to avoid fragile `ExecStart` argument quoting.
- Linux timer first-start hardening to avoid surprise persistent catch-up jobs during setup.
- Bootstrap scripts for Linux and Windows.
- Build scripts for Linux, Windows and PowerShell.
- Debian package layout.
- Arch package layout.
- AppDir/AppImage preparation.
- Project documentation for build, TUI and scheduling workflows.
- `AGENTS.md` for AI coding agents and maintainer instructions.
- GitHub issue templates, pull request template and CI workflow.

### Changed

- Reworked the project from a Windows-only GUI app into a cross-platform backup/export tool.
- Standardized bootstrap scripts under `scripts/`.
- Removed root bootstrap wrapper duplicates.
- Made Arch/EndeavourOS bootstrap safer by avoiding full system upgrades unless `--system-upgrade` is explicitly used.
- Reworked CLI routing and help output.
- Reworked TUI from repeated line-menu output into an alternate-screen dashboard.
- Reworked recurring scheduling through a shared app service layer.
- Updated README for first public release usage.

### Fixed

- Fixed Windows TUI stair-step layout by keeping newline auto-return enabled while using VT escape processing.
- Fixed Windows TUI rendering raw ANSI escape sequences by enabling console virtual terminal processing and falling back to plain text when unavailable.
- Fixed Simkl CLI/TUI auth failure caused by invalid OAuth redirect URI.
- Fixed frontend/Wails type mismatch around schedule settings.
- Fixed stale scheduler unit test expectations.
- Fixed Linux systemd timer setup failure caused by `enable --now` and immediate persistent catch-up.
- Fixed fragile systemd `ExecStart` quoting by generating a wrapper script.
- Fixed recurring backup duplicate-risk through scheduled stale guard.
- Fixed Linux no-GUI behavior by falling back to terminal UI.
- Fixed noisy TUI scrollback behavior.
