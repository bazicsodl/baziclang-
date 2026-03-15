# Bazic Stability Policy (v1)

This policy defines what is considered stable and how breaking changes are handled.

## Semantic Versioning
- `MAJOR.MINOR.PATCH`
- **MAJOR**: breaking changes
- **MINOR**: new features, backward compatible
- **PATCH**: bug fixes and documentation updates

## Stability Guarantees (v1)
The following are stable and will not change in a breaking way in v1:
- Language syntax described in `LANGUAGE_SPEC.md`
- Type system and type inference rules
- Stdlib API signatures in `std/README.md`
- CLI flags documented in `README.md` and `GETTING_STARTED.md`

## Deprecation Process
1. Marked as deprecated in docs and release notes.
2. Warned via lint rules where possible.
3. Removed only in the next major release.

## Runtime Compatibility
- Go backend is the reference implementation.
- LLVM/WASM must match semantics from Go backend for v1 APIs.
- Differences are documented in `COMPATIBILITY_MATRIX.md`.

## Security & Patch Policy
- Security fixes can ship in PATCH releases.
- Severe vulnerabilities may include mitigations or default changes, but no breaking API changes within v1.
