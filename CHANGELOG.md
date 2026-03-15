# Bazic Changelog

## v1.0 (2026-03-12)
- Full-stack web stack: routing via `http_serve_app`, `ServerRequest/ServerResponse`, helpers.
- Auth + security: JWT, bcrypt, session cookies, HMAC/crypto helpers.
- Database stack: SQLite + multi-driver support with params, JSON row encoding, migrations.
- Model tooling: schema + model generator, OpenAPI output.
- LLVM backend parity: JSON getters, auth/session/jwt, HTTP routing server.
- Performance: optimized JSON lookup/encoding, HTTP parsing, benchmarks + gates.
- Supply chain: SBOM generation, lockfile validation, release packaging scripts.

## v0.3 (2026-03-04)
- Language spec freeze (v0.3).
- Stdlib MVP package (`std`).
- LLVM backend parity for MVP features.
- Lockfile v2 with signatures and SBOM.
- LSP, lint, and conformance suite.
- Web (WASM) and desktop MVP targets.
