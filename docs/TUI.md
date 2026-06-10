# Terminal UI

The TUI is launched with:

```bash
SimklExpoGter tui
```

Aliases:

```bash
SimklExpoGter --tui
SimklExpoGter terminal
```

Linux fallback behavior:

- no `DISPLAY` and no `WAYLAND_DISPLAY`: starts TUI automatically
- GUI startup error while attached to a terminal: falls back to TUI
- `--gui`: forces GUI and disables fallback

Controls:

- Type the menu number or command name and press Enter.
- Use `q`, `quit`, or `exit` to leave.
- `Ctrl+C` exits immediately from your shell.
- The UI is deliberately line-oriented, so it works cleanly in SSH, tmux and minimal TTYs.

The TUI supports status, settings, Simkl PIN login, token clearing, easy export, schedule status and recurring backup configuration.


## Authentication

The TUI uses Simkl PIN login. It prints `https://simkl.com/pin/` plus a short code, then polls until approval. This is intentional: Simkl browser OAuth requires an exact registered `redirect_uri`, while PIN login is designed for CLIs and other limited-input devices.


## Full-screen terminal workspace

The TUI uses the terminal alternate screen when attached to a real terminal. The dashboard is redrawn in place instead of printing a fresh menu after every action, so SSH/tmux sessions stay readable.

Controls:

```text
1-7 or command name  select action
Enter                confirm command/input
q                    quit
Ctrl+C               quit and restore the terminal screen
```

The TUI deliberately remains dependency-free and line-input compatible, so it works in plain SSH, tmux, serial consoles and minimal rescue shells.


## Windows console rendering

On Windows, the TUI enables console Virtual Terminal Processing before writing ANSI control sequences. If Windows refuses VT mode, the TUI falls back to plain text and does not use the alternate screen. This prevents raw escape text such as `[?1049h` or `[36m` from appearing in classic console hosts.
