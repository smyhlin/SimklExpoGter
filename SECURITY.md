# Security policy

## Sensitive data

SimklExpoGter can store OAuth/access tokens and optional client secrets in the local user config directory.

Do not publish or attach:

- `settings.json`
- access tokens
- refresh tokens
- Simkl client secrets
- Google Drive client secrets
- exported personal backup files

## Reporting issues

For now, report security issues privately to the repository owner if possible. If private reporting is unavailable, open a minimal public issue without secrets, tokens, logs containing credentials, or exported media history.

## Local file permissions

The app writes settings using user-only file permissions where supported by the OS.
