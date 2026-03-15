# Bazic Language Specification (v1 Draft)

This document defines the stable surface area for Bazic v1.0. It is the authoritative contract for syntax, typing, and runtime behavior. The v1 policy is: **no breaking changes** without a major version bump.

## 1. Lexical Structure
- Source files are UTF-8 text.
- Comments:
  - Line comment: `// ...`
- Identifiers: ASCII letters, digits, and `_`, not starting with a digit.
- Keywords include: `fn`, `let`, `const`, `if`, `else`, `while`, `return`, `struct`, `enum`, `interface`, `impl`, `match`, `import`, `true`, `false`.
- Semicolons are optional; newlines terminate statements unless inside `(...)` or `[...]`.

## 2. Types
### Primitive types
- `int` (signed 64-bit)
- `float` (IEEE 64-bit)
- `bool`
- `string`
- `void`

### Generic types
- `Option[T]`
- `Result[T,E]`

### Structs
```
struct User {
  id: int
  name: string
}
```

### Enums
```
enum Role { Admin, User, Guest }
```

### Interfaces
```
interface Named { fn name(self): string; }
```

## 3. Literals
- `int`: `123`, `-5`
- `float`: `3.14`, `-0.1`
- `bool`: `true`, `false`
- `string`: double-quoted, supports escape sequences (`\n`, `\t`, `\"`)

## 4. Declarations
### Functions
```
fn add(a: int, b: int): int { return a + b }
```

### Global variables
```
const VERSION = "1.0"
let counter = 0
```

### Imports
```
import "std"
import "./local_module"
```

## 5. Statements
- `let`, `const` declarations
- assignment (`name = expr` or `obj.field = expr`)
- `if` / `else`
- `while`
- `return`
- expression statement

## 6. Expressions
- arithmetic: `+ - * / %`
- comparisons: `== != < <= > >=`
- boolean: `&& || !`
- string: `+` for concatenation
- calls: `fn(...)`
- member access: `obj.field`
- method call: `obj.method()` is sugar for `Type_method(obj)` if compatible

## 6.1 Const Rules
- `const` binds an immutable value.
- Assigning to a `const` (including its fields) is a type error.

## 7. Control Flow
### if / else
```
if cond { ... } else { ... }
```

### while
```
while cond { ... }
```

### match
```
match role {
  Role.Admin => { ... }
  Role.User => { ... }
  _ => { ... }
}
```

## 8. Options & Results
Built-in helpers:
- `some(value)` / `none(fallback)`
- `ok(value, err)` / `err(fallback, err)`
- `unwrap_or(opt, fallback)`
- `result_or(res, fallback)`

## 9. Standard Library (v1 Surface)
The stdlib package `std` is stable under v1. See `std/README.md` for full API list.

## 10. Runtime Targets
- Go backend (default)
- LLVM backend (native)
- WASM backend (UI/web)

Compatibility requirements are defined in `COMPATIBILITY_MATRIX.md`.

## 11. Errors
Errors are values (no exceptions). Runtime may abort on unrecoverable conditions; this is documented in `SAFETY.md`.

## 12. SemVer & Stability
See `STABILITY_POLICY.md`.
