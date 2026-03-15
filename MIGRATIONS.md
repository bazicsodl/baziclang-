# Migration Notes

## v1.0
- Web stack: routing via `http_serve_app` with `ServerRequest/ServerResponse`.
- Auth stack: `jwt`, `bcrypt`, session cookies.
- DB stack: drivers, params, JSON row encoding, migrations.
- Model tooling: schema + model generator + OpenAPI output.
- LLVM backend reaches full web/auth/JSON parity.

## v0.3
- Match guards added: `Variant if condition: { ... }` and expression form.
- Generic constraints supported: `fn f[T: Interface](...)`.
- Stdlib MVP available via `import "std";`.
- Lockfile schema upgraded to v2 with signatures and provenance.
- `bazc test` supports `--filter` and `--json`.
- `bazc lint` added.
- LSP server available (`cmd/bazlsp`).

## v0.2 -> v0.3 compatibility notes
- `nil` remains invalid; use `Option`/`Result`/`Error`.
- Lockfile v1 must be upgraded via `bazc pkg sync`.
