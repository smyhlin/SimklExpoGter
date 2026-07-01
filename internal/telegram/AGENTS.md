# AGENTS.md

## Purpose
Telegram Bot API backup upload service.

## Ownership
`internal/telegram/`.

## Local Contracts
- Own Telegram Bot API document upload behavior.
- Keep bot tokens out of logs, errors, and test output.
- Keep Telegram upload separate from Simkl and Google Drive auth.
- Treat Telegram as an optional backup destination, not a required runtime dependency.

## Work Guidance
- Use `sendDocument` for backup files.
- Keep file captions concise and non-secret.
- Add tests around Bot API requests and error handling.

## Verification
- `go test ./internal/telegram`

## Child DOX Index
- None
