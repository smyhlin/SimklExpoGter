# Contributing

Thanks for helping improve SimklExpoGter.

## Development setup

Linux:

```bash
./scripts/bootstrap.sh
./build.sh check
./build.sh linux
```

Windows:

```bat
scripts\bootstrap.bat
build.bat check
build.bat windows
```

## Before opening a pull request

Run the relevant checks:

```bash
./build.sh check
```

For Linux-specific scheduling changes, also verify:

```bash
./build/bin/SimklExpoGter schedule enable --frequency daily --time 02:00 --max-backup-age 24h --format both --field-mode all --content shows,movies,anime
systemctl --user status SimklExpoGterRecurringBackup.timer --no-pager --full
```

## Auth behavior

Use Simkl PIN login for CLI/TUI/headless work. Do not make OAuth redirect login the default for terminal flows.

## Secrets

Never commit:

- Simkl client secrets
- access tokens
- Google Drive tokens
- local settings files
- exported backup files
