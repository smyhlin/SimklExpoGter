# AGENTS.md

## Purpose
Desktop GUI source and configuration for the Wails + Svelte frontend.

## Ownership
`frontend/` configuration, build scripts, and `frontend/src/` UI code.

## Local Contracts
- Keep GUI behavior aligned with the shared service layer.
- Keep Svelte 5 and TypeScript patterns consistent.
- Do not duplicate business logic that belongs in Go services.
- Do not commit generated frontend build output.

## Work Guidance
- Keep labels, status text, and flows in sync with CLI, TUI, and docs.
- Prefer small component changes with clear state flow.
- Keep the UI accessible and cross-platform.

## Verification
- `cd frontend && npm run check`
- `cd frontend && npm run build` when templates or styling change

## Child DOX Index
- None
