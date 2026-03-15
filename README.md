# Bazic Language (AOT Compiler) - Phase 2

Bazic is a compiled language project with a static type checker and native build pipeline.

## Production readiness
See `PRODUCTION_READINESS.md` for the remaining steps to reach full production-grade readiness.
See `LANGUAGE_SPEC.md` and `STABILITY_POLICY.md` for v1 behavior guarantees.
See `STDLIB_SECURITY.md` for safe-usage notes across stdlib APIs.
See `PERFORMANCE_PLAN.md` and `DISTRIBUTION_PLAN.md` for perf and release planning.
See `BAZIC_UI_FRONTEND_V1.md` and `BAZIC_UI_TOOLKIT.md` for the Bazic UI frontend contract and workflow.

## Current capabilities
- AOT compilation: `.bz` -> generated Go -> native binary/wasm
- Multi-file modules via `import "...";`
- `struct` declarations and struct literals
- Generic structs: `struct Box[T] { ... }`
- `enum` declarations
- `interface` + `impl` conformance checks
- Method-call syntax: `u.label()` with compatibility for `User_label(u)` convention
- Exhaustive enum `match` statement and expression
- Generic functions (`fn identity[T](x: T): T { ... }`)
- Generic constraints (`fn f[T: Named](x: T): string { ... }`)
- Safety prelude types available in every program:
  - `Error`, `Option[T]`, `Result[T,E]`
- Safety helpers available in every program:
  - `some(value)`, `none(fallback)`, `ok(value, fallbackErr)`, `err(fallbackValue, errValue)`
- Error-flow helpers:
  - `unwrap_or(opt, fallback)`, `result_or(res, fallback)`
- Stdlib MVP package (`import "std";`) with:
  - `std/io`, `std/fs`, `std/time`, `std/json`, `std/http`, `std/crypto`, `std/base64`, `std/collections`, `std/db`, `std/os`, `std/path`
  - `std/http` supports `HttpOptions` (timeouts, headers, user agent, content type, TLS options), `HttpRequest` (custom method), and `HttpResponse` (status, headers, body). Headers are newline-separated `"Key: Value"` entries.
- Standard builtins available in every program:
  - `str`, `len`, `contains`, `starts_with`, `ends_with`, `to_upper`, `to_lower`, `trim_space`, `replace`, `repeat`
  - `parse_int`, `parse_float`
- Explicit null policy: `nil` is rejected; use `Option`/`Result`/`Error`
- Static typing + local inference for `let` and `const`
- Control flow: `if`, `else`, `while`, `return`
- Semicolons optional; newlines terminate statements
- LLVM backend (early): `emit-llvm` now emits real non-generic function signatures, arithmetic/comparison/logical return-expression IR, direct non-generic calls, control-flow lowering for `let`/assign/if/while/return, enum `match`, non-generic struct layout/field access, monomorphization for generic structs/functions, string ops/stdlib builtin lowering, and interface type lowering (with `print`/`println` lowered via `printf`)

## CLI
```powershell
.\tools\install.ps1
.\bin\bazic.exe version
.\bin\bazic.exe new hello
cd hello
.\bin\bazic.exe run
.\bin\bazic.exe build  # outputs to .\bin\hello.exe by default
.\bin\bazic.exe check .\examples\phase2\main.bz
.\bin\bazic.exe run .  # directory => ./main.bz
.\bin\bazic.exe doctor
.\bin\bazic.exe run . -- arg1 arg2
.\bin\bazic.exe clean
.\bin\bazic.exe repl
.\bin\bazic.exe check .\examples\phase3\main.bz
.\bin\bazic.exe check --compat v1.0 .\examples\phase3\main.bz
.\bin\bazic.exe run .\examples\phase3\main.bz
.\bin\bazic.exe run --backend go .\examples\phase3\main.bz
.\bin\bazic.exe build .\examples\phase3\main.bz -o .\examples\phase3\app.exe
.\bin\bazic.exe transpile .\examples\phase2\main.bz -o .\examples\phase2\generated.go
.\bin\bazic.exe fmt .\examples\phase3\main.bz
.\bin\bazic.exe fmt --check .\examples\phase3\main.bz
.\bin\bazic.exe fmt --check .\examples
.\bin\bazic.exe fmt --stdout .\examples\phase3\main.bz
.\bin\bazic.exe test .\examples\phase3
.\bin\bazic.exe test --filter match .\examples\phase3
.\bin\bazic.exe test --json .\examples\phase3
.\bin\bazic.exe test --backend go .\examples\phase3
.\bin\bazic.exe test --backend all .\examples\phase3
.\bin\bazic.exe lint .
.\bin\bazic.exe build .\examples\\web\\app.bz --target wasm -o .\\examples\\web\\app.wasm
.\bin\bazic.exe build .\examples\\apps\\desktop\\main.bz -o .\\examples\\apps\\desktop\\app.exe
.\bin\bazic.exe ui build --dir .\examples\web
.\bin\bazic.exe ui dev --dir .\examples\web --port 8080
.\bin\bazic.exe ui init --dir .\my-ui
.\bin\bazic.exe ui init --dir .\my-ui
.\bin\bazic.exe ui page settings --dir .\my-ui
.\bin\bazic.exe ui component card --dir .\my-ui
.\bin\bazic.exe ui build --dir .\my-ui
.\bin\bazic.exe ui dev --dir .\my-ui --port 8080
.\bin\bazic.exe ui layout --dir .\my-ui
.\bin\bazic.exe ui routes --dir .\my-ui
.\bin\bazic.exe ui migrate-layout --dir .\my-ui
.\bin\bazic.exe ui init --dir .\my-ui-react --template react
.\bin\bazic.exe ui init --dir .\my-ui-react-ts --template react-ts
.\bin\bazic.exe emit-llvm .\examples\phase3\main.bz -o .\examples\phase3\main.ll
.\bin\bazic.exe emit-llvm --check .\examples\phase3\main.bz
.\bin\bazic.exe run .\examples\apps\cli\main.bz

# Dev compiler (Go backend default)
go run .\cmd\bazc\ version
go run .\cmd\bazc\ run .\examples\phase3\main.bz
```
Note: `bazic` defaults to the LLVM backend (requires `clang` on PATH). `bazic build --target wasm` (or building a web target under `examples/web`) automatically switches to the Go backend; otherwise use `--backend go` explicitly.
Note: LLVM builds look for `runtime/bazic_runtime.c` in the project root, or under `BAZIC_HOME/runtime`, or next to the Bazic install.
Note: `bazic new`/`bazic init` auto-add the stdlib if `BAZIC_STDLIB` or `BAZIC_HOME` is set, or if a `std` directory is adjacent to the `bazic` binary.
Tip: `bazic doctor` reports toolchain and stdlib status.
Note: `bazic new`/`bazic init` ensure `.gitignore` contains Bazic build outputs (`.bazic/`, `bin/`, `*.exe`, `*.wasm`).
Tip: LLVM builds honor `BAZIC_CLANG` (compiler path) and `BAZIC_CLANG_FLAGS` (extra flags).
Tip: Go/WASM builds set `BAZIC_TARGET=wasm` internally to avoid incompatible DB drivers.
Tip: Set `BAZIC_BACKEND=go` to force the Go backend by default (useful if LLVM/clang is not installed).

Linux/macOS install:
```bash
./tools/install.sh
```
Installer copies `std` next to the `bazic` binary so new projects auto-wire the standard library.
Installer also installs `bazlsp`, `runtime/`, and the Bazic VS Code extension (auto-installs if `code` is on PATH), and adds Bazic to `PATH` and `BAZIC_HOME`.
Unsigned releases will trigger Windows SmartScreen warnings; verify `SHA256SUMS.txt` until code signing is enabled.
Windows users can run `Get-FileHash -Algorithm SHA256 .\dist\packages\bazic-windows-amd64.msi` and compare with `dist/packages/SHA256SUMS.txt`.

## Package manager workflow (local path dependencies)
```powershell
cd C:\Users\Ipeh\Documents\baziclang
.\bin\bazic.exe pkg init baziclang
.\bin\bazic.exe pkg add stdutil .\examples\packages\stdutil
.\bin\bazic.exe pkg sync
.\bin\bazic.exe pkg verify
.\bin\bazic.exe pkg sbom -o .\bazic.sbom.json
```

This creates:
- `bazic.mod.json`
- `bazic.lock.json`
- `.bazic/pkg/<alias>/...`

Then import packages by alias:
```bazic
import "stdutil";
```

Stdlib MVP uses alias `std`:
```bazic
import "std";
```
Security defaults:
- Absolute imports are disallowed.
- Alias imports are resolved from `.bazic/pkg/<alias>` only.
- `bazic.lock.json` checksums are verified at import resolution.
- If package cache is tampered or stale, compiler asks for `bazc pkg sync`.
- `bazc pkg verify` checks manifest/lock/cache consistency for CI/release gates.
- Lockfile v2 stores provenance + integrity metadata and is signed with an ed25519 project key.
- `bazc pkg verify` enforces signature + checksum + provenance policy.
- `bazc pkg sbom` writes a minimal SBOM (`bazic.sbom.json`) with dependency provenance and checksums.
- Local dependency source drift (source changed after sync) is detected and rejected until re-sync.
- Compiler entry commands (`check/run/build/transpile/emit-llvm`) automatically run package integrity verification when a project manifest is present.
- Lexer/parser failures now include source snippets with line/column caret diagnostics.
- `bazc fmt [path]` provides canonical formatting for Bazic source (`.bz` file or directory).
- `bazc test [path]` runs Bazic source tests (`*_test.bz`, `fn test_*(): bool|void`).
- `bazc test --filter <name>` runs only tests matching substring.
- `bazc test --json` emits machine-readable results.
- `bazc lint [path]` runs `bazlint` correctness/security/style checks.
- `assert(cond: bool)` and `assert_msg(cond: bool, msg: string)` are available via prelude and integrated with `bazc test`.
- Unused local variables/parameters are compile errors (`_` is allowed as discard).
- Non-`void` functions are required to return on all control paths.
- `bazc build` uses deterministic Go flags by default (`-trimpath`, empty buildid).
- Entrypoint is strict: `fn main(): void` (no params, no generics).
- Unknown symbol diagnostics include typo suggestions when available.
- Import cycles are rejected with explicit cycle chains in diagnostics.

## Project layout
- `cmd/bazic` default CLI (LLVM backend by default)
- `cmd/bazc` compiler CLI (Go backend by default)
- `cmd/bazlsp` Bazic language server (LSP)
- `internal/lexer` tokenizer
- `internal/parser` parser
- `internal/sema` static type checker
- `internal/codegen` Go backend
- `internal/bazlint` lint rules
- `internal/compiler` import graph loader + build orchestration
- `internal/pkgm` module manifest and dependency sync
- `examples` runnable language samples
- `examples/web` wasm + JS interop demo
- `examples/apps/desktop` desktop MVP demo
- `conformance` Bazic conformance test suite
- `tools/vscode` VS Code extension (LSP client)
- `legacy_python` archived prototype

## Tests
```powershell
go test ./...
.\bin\bazic.exe lint .
.\bin\bazic.exe test .\conformance
.\bin\bazic.exe test .\conformance --backend all
.\bin\bazic.exe build .\examples\\web\\app.bz --target wasm -o .\\examples\\web\\app.wasm
.\bin\bazic.exe build .\examples\\apps\\desktop\\main.bz -o .\\examples\\apps\\desktop\\app.exe
go test .\internal\\compiler -run ^$ -bench . -benchmem
```

To run conformance as part of Go tests:
```powershell
$env:BAZIC_RUN_CONFORMANCE=1; go test .\internal\\testrun -run ConformanceSuite
```

## Platform Targets
- Web (WASM): `tools/web/build.ps1` or `tools/web/build.sh`, then serve `examples/web`.
- Desktop: `.\bin\bazic.exe build .\examples\\apps\\desktop\\main.bz -o .\\examples\\apps\\desktop\\app.exe`
- Desktop packaging: `.\tools\\desktop\\package.ps1` or `./tools/desktop/package.sh`
- LLVM native (requires clang on PATH): `.\bin\bazic.exe build .\examples\\phase3\\main.bz -o .\\examples\\phase3\\app.exe`

## LLVM Native Runtime Notes
- The LLVM backend links a minimal C runtime at `runtime/bazic_runtime.c`.
- LLVM builds require clang 15+ (opaque pointers). On Windows, install Visual Studio Build Tools (C++ workload / Windows SDK) so clang can find standard C headers.
- Supported: `io_read_line`, `fs_*`, `time_*`, `json_escape`, `crypto_sha256_hex`, `crypto_random_hex`, `desktop_open_url`.
- HTTP on Windows: `http_get`, `http_post`, `http_serve_text`.
- HTTP on non-Windows requires `libcurl` dev package and links `-lcurl`.
- HTTP defaults: 5s connect timeout, 15s total timeout, `User-Agent: Bazic/1.0`, `Accept: */*`, and POST uses `Content-Type: text/plain; charset=utf-8`.
- TLS options: `tls_insecure_skip_verify` and `tls_ca_bundle_pem` are supported on non-Windows; Windows does not support custom CA bundle.
- SQLite: `std/db` uses SQLite via the Go backend (pure Go driver). Native LLVM runtime DB support is available with `BAZIC_SQLITE=1` and `sqlite3` dev library (links `-lsqlite3`). Native also accepts `db_exec_with("sqlite" | "sqlite3", dsn, ...)`.
- Other DBs (Go backend): use `db_exec_with` / `db_query_with` with drivers like `postgres` and `mysql` (aliases: `postgresql`).
- `crypto_random_hex` uses a cryptographically secure RNG in the native runtime (BCrypt on Windows, arc4random on BSD/macOS, /dev/urandom on Linux).

## Release And Security
- `RELEASE.md` for release process.
- `SECURITY.md` for security policy.
- `MIGRATIONS.md` for migration notes.
- `BENCHMARKS.md` for compiler + runtime performance benchmarks (use `scripts/bench.ps1` for runtime).
- `SAFETY.md` for Bazic safety model and `any` usage policy.
- `GETTING_STARTED.md` for the shortest setup/run path.
- `V1_GUIDE.md` for the v1 documentation structure.
- `INTEGRATIONS.md` for integration roadmap (DBs, gRPC, brokers, plugin/FFI).
- `MIGRATION_TOOLING.md` for planned migration tooling and compatibility gates.
- `REFERENCE_APPS.md` for the v1 reference apps list.
- `READINESS_CHECKLIST.md` for the v1.0 ship checklist.
- `BAZIC_UI.md` for Bazic UI and web app scaffolding.
- `BAZIC_UI_GUIDE.md` for a Bazic UI walkthrough.
- `tools/release/build.ps1` and `tools/release/build.sh` for release builds.

## Editor Support
- LSP server: `bazlsp` (preferred) or `go run .\cmd\bazlsp`
- VS Code client: `tools/vscode` (see `tools/vscode/README.md`)
  - Auto-format and preferred quick fixes on save.

## CI Quality Gates
- GitHub Actions workflow: `.github/workflows/ci.yml`
- Gates:
  - `go test ./...`
  - `bazc pkg sync` + `bazc pkg verify`
  - `bazc pkg sbom`
  - Bazic `check` for phase3 examples
  - Bazic `run/build/emit-llvm` smoke tests
  - `bazc lint` and `bazc test ./conformance`
  - LLVM backend conformance + build smoke tests

## Example
See [examples/phase2/main.bz](./examples/phase2/main.bz) and [examples/phase2/lib/main.bz](./examples/phase2/lib/main.bz).
See [examples/phase3/main.bz](./examples/phase3/main.bz) for generic structs + interface impl.
See [examples/phase3/safety.bz](./examples/phase3/safety.bz) for `Option`/`Result`/`Error` usage.
See [examples/phase3/match.bz](./examples/phase3/match.bz) for exhaustive enum `match`.
See [examples/phase3/match_expr.bz](./examples/phase3/match_expr.bz) for expression-style `match`.
See [examples/phase3/sample_test.bz](./examples/phase3/sample_test.bz) for `bazc test` conventions.
See [std/README.md](./std/README.md) for stdlib MVP APIs and examples.
See [examples/apps/cli/main.bz](./examples/apps/cli/main.bz) and [examples/apps/service/main.bz](./examples/apps/service/main.bz) for stdlib-powered reference apps.
