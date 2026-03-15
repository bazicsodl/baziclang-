# Security Policy

## Supported Versions
- `nightly`, `beta`, and latest `stable` are supported.

## Reporting a Vulnerability
- Email: security@baziclang.local
- Please include a proof-of-concept, impact analysis, and suggested fix if possible.

## Disclosure Process
1. Triage within 72 hours.
2. Fix and validate with tests and reproducibility checks.
3. Coordinate disclosure timeline with reporters.
4. Publish security advisory and patch release.

## Supply Chain Security
- All dependencies are recorded in `bazic.lock.json` (v2).
- Lockfile entries are signed; `bazc pkg verify` enforces signatures.
- SBOM available via `bazc pkg sbom`.

## Reproducible Builds
- Deterministic build flags are used by default.
- See `tools/repro/verify.ps1` or `tools/repro/verify.sh`.

## Runtime Hardening Defaults (Go backend)
- HTTP server sets read/write/idle timeouts and caps request body size.
- See `WEB_STACK.md` for environment overrides.

## Fuzzing
- Parser and lexer fuzz tests are included under `internal/lexer` and `internal/parser`.
- HTTP handler name parsing fuzz tests are included under `internal/codegen`.
- Model schema JSON parsing fuzz tests are included under `internal/modelgen`.

## Stdlib Guidance
See `STDLIB_SECURITY.md` for safe-usage notes across HTTP, DB, crypto, and filesystem APIs.
