# AGENTS.md

## Purpose
GitHub workflows and repository templates.

## Ownership
`.github/workflows/*.yml`, `.github/PULL_REQUEST_TEMPLATE.md`, and related GitHub-side repository metadata.

## Local Contracts
- Keep CI and release workflows aligned with the build scripts and release notes policy.
- Keep templates concise and current.
- Do not move build logic into workflows unless GitHub Actions specifically needs it.

## Work Guidance
- Update workflows when build entrypoints, release artifacts, or validation steps change.
- Keep workflow names, triggers, and job responsibilities explicit.
- Keep templates aligned with the current contribution process.

## Verification
- Review workflow syntax and match it against the current build commands

## Child DOX Index
- None
