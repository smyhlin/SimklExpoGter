# Recurring backups

SimklExpoGter uses OS-native schedulers instead of a long-running background daemon.

## Windows

Windows uses Task Scheduler.

The fixed task name is:

```text
SimklExpoGterRecurringBackup
```

The task launches the built executable with the `run` command and saved options.

## Linux

Linux uses a systemd user timer.

Unit files are written to:

```text
~/.config/systemd/user/
```

Generated unit names are based on:

```text
SimklExpoGterRecurringBackup.service
SimklExpoGterRecurringBackup.timer
```

Useful commands:

```bash
systemctl --user list-timers '*SimklExpoGter*'
systemctl --user status SimklExpoGterRecurringBackup.timer
systemctl --user status SimklExpoGterRecurringBackup.service
journalctl --user -u SimklExpoGterRecurringBackup.service
```

For timers to run without an active login session, the app now tries to enable lingering automatically during `schedule enable`. If the OS policy blocks that, run:

```bash
SimklExpoGter schedule linger
# or manually:
loginctl enable-linger "$USER"
```

## macOS

macOS scheduling is not implemented yet.


## Stale backup guard

Scheduled backups use a stale-backup guard by default.

The scheduler launches:

```bash
SimklExpoGter run --scheduled
```

Before exporting, the app checks the last successful backup timestamp saved in `settings.json`:

```json
{
  "lastSuccessfulBackupAt": "2026-06-09T02:00:00Z",
  "lastSuccessfulBackupKind": "scheduled"
}
```

If the last successful backup is newer than the configured threshold, the scheduled run exits successfully without creating duplicate files. If the backup is missing or older than the threshold, the backup runs and updates the timestamp only after a successful export/upload. Saved backup storage may be local, Google Drive, or Telegram.

Supported threshold examples:

```text
12h
24h
3d
1w
```

The default threshold is `24h`.

CLI examples:

```bash
SimklExpoGter schedule enable --frequency daily --time 02:00 --max-backup-age 24h
SimklExpoGter schedule enable --frequency weekly --days mon,thu --time 03:30 --max-backup-age 3d
SimklExpoGter run --scheduled
SimklExpoGter run --scheduled --max-backup-age 12h
```

Disable the stale guard and always run whenever the OS scheduler triggers:

```bash
SimklExpoGter schedule enable --no-stale-guard
```

## Linux boot/offline behavior

Linux timers are installed as systemd user timers with:

```ini
Persistent=true
```

That means a missed timer can run after the system/user manager becomes available again.

For timers to work without an active graphical login session, enable lingering:

```bash
SimklExpoGter schedule linger
```

Equivalent manual command:

```bash
loginctl enable-linger "$USER"
```

## Linux timer implementation notes

The Linux scheduler now writes three files to `~/.config/systemd/user/`:

```text
SimklExpoGterRecurringBackup.timer
SimklExpoGterRecurringBackup.service
SimklExpoGterRecurringBackup-run.sh
```

The service calls the wrapper script instead of embedding every CLI argument directly into `ExecStart`. This avoids fragile systemd command-line quoting around paths and arguments.

During schedule setup the timer is started once with `Persistent=false`, then the timer file is rewritten with `Persistent=true`. This avoids an immediate catch-up backup during configuration while still keeping missed-run catch-up behavior for future boots/logins.

If scheduling fails, inspect:

```bash
systemctl --user status SimklExpoGterRecurringBackup.timer --no-pager --full
systemctl --user status SimklExpoGterRecurringBackup.service --no-pager --full
journalctl --user -u SimklExpoGterRecurringBackup.service -n 80 --no-pager
cat ~/.config/systemd/user/SimklExpoGterRecurringBackup-run.sh
```


## Automatic lingering

On Linux, `schedule enable` attempts to enable lingering with `loginctl enable-linger "$USER"` after installing the user timer. This lets the user manager exist after logout and at boot. Failure to enable lingering does not block timer installation, but the status message tells the user what to run manually.
