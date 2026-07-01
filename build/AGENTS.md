# AGENTS.md

## Purpose
Build assets and build-time documentation for the project.

## Ownership
`build/` files that support Wails desktop builds, Windows resources, macOS plist assets, and build notes.

## Local Contracts
- Keep build assets cross-platform and in sync with `build.sh`, `build.bat`, and `build.ps1`.
- Do not commit generated binaries, bundles, or package outputs.
- Keep `build/README.md` accurate for the files in this directory.
- Preserve the Wails asset layout used by the current build scripts.

## Work Guidance
- Update icons, manifests, and plist files together with the scripts that consume them.
- Treat `build/bin/` as generated output.
- Keep platform-specific files minimal and explicit.

## Verification
- `./build.sh check`
- Relevant build target for the file changed

## Child DOX Index
- None
