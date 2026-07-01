# AGENTS.md

## Purpose
Linux packaging assets and distro-specific metadata.

## Ownership
`packaging/` files for AppImage, Arch, Debian, and desktop integration.

## Local Contracts
- Keep packaging metadata in sync with current build outputs and app naming.
- Keep desktop entry, PKGBUILD, AppRun, and Debian control files aligned.
- Do not commit generated archives or package trees.

## Work Guidance
- Update packaging metadata alongside build-script changes.
- Keep Linux desktop integration explicit and cross-distro.
- Match package dependencies to the actual runtime requirements.

## Verification
- Relevant build targets such as `./build.sh deb`, `./build.sh arch`, or `./build.sh appimage`

## Child DOX Index
- None
