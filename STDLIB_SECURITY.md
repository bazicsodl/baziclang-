# Bazic Stdlib Security Notes

This document summarizes security considerations and safe defaults for Bazic stdlib APIs.

## HTTP
- Always set explicit timeouts (server defaults are applied by `http_serve_app`).
- Avoid echoing raw request bodies without validation.
- Validate all user input before using it in queries or file paths.

## Database
- Prefer parameterized queries using `db_*_params` helpers.
- Never build SQL with direct string concatenation for untrusted input.
- Use least-privileged DB credentials in production.

## Crypto
- Use `crypto_bcrypt_hash` for password storage; never store plaintext.
- Use `crypto_random_hex` for secure token generation.
- JWT tokens should be short-lived and signed with a strong secret.

## Auth & Sessions
- Always set `secure=true` for cookies in production.
- Rotate session secrets when deploying.
- Invalidate sessions on password reset or privilege changes.

## JSON
- Treat all decoded JSON as untrusted input.
- Use `json_validate` before parsing complex payloads.

## Filesystem
- Avoid writing user-controlled paths.
- Use `path_join` and validate final paths to prevent traversal.

## Environment
- Secrets should only come from environment variables or a vault.
- Never log secrets or raw credentials.
