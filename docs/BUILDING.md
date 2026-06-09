# Building SimklExpoGter

## Quick start

Linux:

```bash
./scripts/bootstrap.sh
./build.sh check
./build.sh linux
./build.sh linux-cli
```

On Arch/EndeavourOS, `scripts/bootstrap.sh` avoids full system upgrades by default and installs only missing packages with `pacman -S --needed`. Use `./scripts/bootstrap.sh --system-upgrade` only when you intentionally want it to run `pacman -Syu` too.

Windows:

```bat
scripts\bootstrap.bat
build.bat check
build.bat windows
build.bat windows-cli
```

## Linux targets

`./build.sh linux` builds the normal Wails GUI binary.

`./build.sh linux-cli` builds a smaller CLI/TUI-only binary with `go build -tags cli`. This target does not compile `main.go` or `app.go`, so it avoids Wails GUI startup code and is better for servers, SSH and guiless machines.

`./build.sh deb` builds a Debian package from the Linux GUI binary.

`./build.sh appimage` builds the Linux GUI binary, prepares an AppDir and runs `linuxdeploy` when it is available.

`./build.sh arch` builds a native Arch package when `bsdtar` is available, otherwise it prepares a package tree and copies the reference `PKGBUILD` into `dist/arch`.

## Windows targets

`build.bat windows` builds the Wails GUI `.exe`.

`build.bat windows-cli` builds the CLI/TUI-only `.exe` with `go build -tags cli`.

`build.ps1` is a PowerShell wrapper around `build.bat`.

## WebKitGTK tag handling

Wails v2 normally uses WebKitGTK 4.0 on Linux. Newer Debian/Ubuntu releases may only provide the 4.1 development package. `scripts/bootstrap.sh` detects this with `pkg-config` and writes `.build/webkit.env`.

`build.sh` sources that file and adds `-tags webkit2_41` to Wails builds when required.

## AppImage note

The AppImage target is best-effort because Wails v2 GUI binaries depend on GTK/WebKitGTK. Native `.deb` and Arch packages are cleaner for real installs. Use AppImage when you need a portable artifact and test it on the target distro.


## Scheduler CLI smoke test

After building a real binary, verify scheduler commands:

```bash
./dist/linux/SimklExpoGter-cli schedule status
./dist/linux/SimklExpoGter-cli schedule enable --frequency daily --time 02:00 --max-backup-age 24h
./dist/linux/SimklExpoGter-cli run --scheduled
./dist/linux/SimklExpoGter-cli schedule linger
```

On Linux, `schedule linger` enables systemd user services after logout/boot through `loginctl enable-linger "$USER"`.
