# Bazic Compatibility Matrix (v1 Draft)

This matrix tracks platform and backend coverage.

## Backends
- Go backend: production-ready
- LLVM backend: production-ready after conformance pass

## Targets
| Target | Go Backend | LLVM Backend | Notes |
| --- | --- | --- | --- |
| Windows (x64) | ✅ | ✅ | Primary dev/test platform |
| Linux (x64) | ✅ | ✅ | CI recommended |
| macOS (x64/arm64) | ✅ | ✅ | CI recommended |
| WASM (browser) | ✅ | ❌ | Go backend only |

## Runtime Feature Coverage
- IO/FS/JSON/HTTP/crypto: ✅
- DB (SQLite optional): ✅ (with `BAZIC_SQLITE=1`)
- Web/WASM interop: ✅ (Go backend)
- Desktop open URL: ✅

## Notes
- LLVM backend now passes the conformance suite.
- WASM target remains tied to the Go backend.
