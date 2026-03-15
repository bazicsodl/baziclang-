# Bazic v1 Guide

This guide is the v1 reference for Bazic users and teams shipping real systems.

## 1. Overview
Bazic is a compiled, statically typed language designed for small surface area, strict safety defaults, and predictable performance. It targets:
- **Backend services** (native binaries).
- **Web** (WASM via Go backend).
- **Desktop** (native, simple GUI/open‑url patterns).

Non‑goals:
- Dynamic typing in core logic.
- Implicit nulls or hidden error flow.

## 2. Quickstart
Install and run:
```powershell
.\tools\install.ps1
.\bin\bazic.exe new hello
cd hello
.\bin\bazic.exe run
```

Tests and lint:
```powershell
.\bin\bazic.exe test .
.\bin\bazic.exe lint .
```

## 3. Language Basics
Variables are declared with `let` and are block‑scoped:
```bazic
let x = 3;
let s: string = "hi";
```

Functions are declared with explicit parameter and return types:
```bazic
fn add(a: int, b: int): int {
    return a + b;
}
```

Control flow:
```bazic
if a > b { println("a"); } else { println("b"); }
while i < 10 { i = i + 1; }
```

## 4. Structs, Enums, and Match
```bazic
struct User { name: string; age: int; }
enum Role { Guest, Admin }

fn is_admin(r: Role): bool {
    match r {
        Guest: { return false; }
        Admin: { return true; }
    }
}
```

`match` is exhaustive for enums. Guarded arms do not satisfy exhaustiveness.

## 5. Generics and Interfaces
```bazic
struct Box[T] { value: T; }
interface Named { fn label(self: User): string; }
impl User: Named;

fn identity[T](x: T): T { return x; }
```

Bounds require the type to implement the interface:
```bazic
struct Boxed[T: Named] { value: T; }
```

## 6. Error Handling
Use `Result` for fallible operations and `Option` for absence:
```bazic
let r = parse_int("123");
if r.is_ok { println(str(r.value)); }

let o = some(7);
println(str(unwrap_or(o, 0)));
```

`nil` is rejected at compile time.

## 7. Standard Library
Import `std` to access modules:
```bazic
import "std";
```

Highlights:
- `std/fs`: file IO
- `std/time`: time helpers
- `std/json`: validation, minify, pretty
- `std/http`: client + simple server
- `std/crypto`: sha256 + random hex
- `std/collections`: StringBuilder

## 8. Tooling
```powershell
.\bin\bazic.exe fmt .
.\bin\bazic.exe test .\conformance
.\bin\bazic.exe emit-llvm --check .\main.bz
```

## 9. Targets
- **Native (LLVM)**: `bazic build --backend llvm`
- **Web (WASM)**: `bazic build --target wasm`
- **Desktop**: `examples/apps/desktop`

## 10. Performance
Use `scripts/bench.ps1` and `scripts/bench_gate.ps1` for runtime benches and regression gates.

## 11. Safety Model
See `SAFETY.md` for `any` policy and unsafe boundary rules.

## 12. Packages
Use `bazic pkg sync/verify/sbom` for supply‑chain validation and reproducible builds.

## 13. Migration and Stability
See `LANGUAGE.md` compatibility policy and `MIGRATIONS.md` for deprecations.

## 14. Docs & Policies
- `RELEASE.md`
- `SECURITY.md`
- `MIGRATION_TOOLING.md`
- `V1_STABILITY.md`
- `COMPATIBILITY_MATRIX.md`
