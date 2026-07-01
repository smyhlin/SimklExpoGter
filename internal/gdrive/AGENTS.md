# AGENTS.md

## Purpose
Google Drive OAuth and backup upload service.

## Ownership
`internal/gdrive/`.

## Local Contracts
- Own Google Drive OAuth configuration, PKCE exchange, token refresh, folder creation, and file upload.
- Keep Google Drive auth separate from Simkl auth.
- Preserve the local-temp upload flow and explicit token handling.

## Work Guidance
- Keep credentials and tokens out of logs and error text.
- Make redirect URI and PKCE requirements explicit in code paths that use them.
- Keep upload results stable for callers that persist folder metadata.

## Verification
- `go test ./internal/gdrive`

## Child DOX Index
- None
