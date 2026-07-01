# AGENTS.md

## Purpose
Repository bootstrap and setup scripts.

## Ownership
`scripts/` shell, batch, and PowerShell helpers.

## Local Contracts
- Keep shell, batch, and PowerShell entrypoints behaviorally aligned.
- Keep bootstrap scripts cross-platform and repository-local.
- Do not hard-code machine-specific assumptions.

## Work Guidance
- Keep option names and output text consistent across script variants.
- Preserve bootstrap behavior for dependencies, Wails, and frontend setup.
- Keep the scripts copy-paste friendly.

## Verification
- `bash -n scripts/bootstrap.sh`
- `./scripts/bootstrap.sh --help` or platform equivalent after edits

## Child DOX Index
- None
