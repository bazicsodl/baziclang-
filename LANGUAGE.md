# Bazic Language Specification (v1.0)

Status: Stable. This document is the normative specification for Bazic v1.0 unless marked as non-normative. If a behavior is described here, implementations must follow it.

## 1. Compatibility Policy
1. v1.x releases do not introduce breaking changes; breaking changes require a v2.0 release.
2. Deprecations require clear migration notes and at least one minor release before removal.
3. The following are guaranteed stable within v1.x: grammar, core typing rules, entrypoint contract, match rules, null policy, import resolution, and package integrity enforcement.
4. The following may still evolve in minor releases: standard library surface (additive only), diagnostics formatting, LLVM lowering details.

## 2. Lexical Structure
1. Files are UTF-8 text. Line endings may be LF or CRLF.
2. Tokens include identifiers, keywords, literals, punctuation, and operators.
3. Identifiers match `[A-Za-z_][A-Za-z0-9_]*` and are case-sensitive.
4. Whitespace separates tokens and is otherwise insignificant.
5. Comments are supported:
   - Line comment: `// ...` to end of line.
   - Block comment: `/* ... */` (not nested).
6. String literals support escapes: `\\`, `\"`, `\n`, `\r`, `\t`.
7. Semicolons are optional; newlines terminate statements unless inside `(...)` or `[...]`.

## 3. Grammar (EBNF)
This grammar is complete for the current language features. If a construct is not described here, it is not part of v1.0.

Program         = { Decl } ;

Decl            = ImportDecl
               | StructDecl
               | EnumDecl
               | InterfaceDecl
               | ImplDecl
               | FuncDecl
               | GlobalLetDecl
               ;

ImportDecl      = "import" String [ ";" ] ;

StructDecl      = "struct" Ident [ TypeParams ] "{" { StructField } "}" ;
StructField     = Ident ":" Type [ ";" ] ;

EnumDecl        = "enum" Ident "{" Ident { "," Ident } "}" ;

InterfaceDecl   = "interface" Ident "{" { InterfaceMethod } "}" ;
InterfaceMethod = "fn" Ident "(" [ Params ] ")" ":" Type [ ";" ] ;

ImplDecl        = "impl" Type ":" Ident [ ";" ] ;

FuncDecl        = "fn" Ident [ TypeParams ] "(" [ Params ] ")" ":" Type Block ;

GlobalLetDecl   = ("let" | "const") Ident [ ":" Type ] "=" Expr [ ";" ] ;

TypeParams      = "[" TypeParam { "," TypeParam } "]" ;
TypeParam       = Ident [ ":" Ident ] ;
Params          = Param { "," Param } ;
Param           = Ident ":" Type ;

Type            = BuiltinType | Ident | GenericType ;
GenericType     = Ident "[" Type { "," Type } "]" ;
BuiltinType     = "int" | "float" | "bool" | "string" | "void" | "any" ;

Block           = "{" { Stmt } "}" ;

Stmt            = LetStmt
               | AssignStmt
               | IfStmt
               | WhileStmt
               | MatchStmt
               | ReturnStmt
               | ExprStmt
               ;

LetStmt         = ("let" | "const") Ident [ ":" Type ] "=" Expr [ ";" ] ;
AssignStmt      = AssignTarget "=" Expr [ ";" ] ;
AssignTarget    = Ident { "." Ident } ;
IfStmt          = "if" Expr Block [ "else" Block ] ;
WhileStmt       = "while" Expr Block ;
MatchStmt       = "match" Expr "{" { MatchArm } "}" ;
MatchArm        = Ident [ "if" Expr ] ":" Block ;
ReturnStmt      = "return" [ Expr ] [ ";" ] ;
ExprStmt        = Expr [ ";" ] ;

Expr            = MatchExpr
               | BinaryExpr
               ;

MatchExpr       = "match" Expr "{" MatchExprArm { "," MatchExprArm } [ "," ] "}" ;
MatchExprArm    = Ident [ "if" Expr ] ":" Expr ;

BinaryExpr      = UnaryExpr { BinaryOp UnaryExpr } ;
UnaryExpr       = UnaryOp UnaryExpr | PrimaryExpr ;

PrimaryExpr     = Literal
               | Ident
               | CallExpr
               | MethodCallExpr
               | FieldAccessExpr
               | StructLitExpr
               | "(" Expr ")"
               ;

CallExpr        = Ident "(" [ Args ] ")" ;
MethodCallExpr  = PrimaryExpr "." Ident "(" [ Args ] ")" ;
FieldAccessExpr = PrimaryExpr "." Ident ;
StructLitExpr   = Ident [ TypeArgs ] "{" StructLitField { "," StructLitField } [ "," ] "}" ;
StructLitField  = Ident ":" Expr ;

Args            = Expr { "," Expr } ;
TypeArgs        = "[" Type { "," Type } "]" ;

Literal         = IntLit | FloatLit | BoolLit | String ;
BoolLit         = "true" | "false" ;

UnaryOp         = "-" | "!" ;
BinaryOp        = "+" | "-" | "*" | "/" | "%"
               | "==" | "!=" | "<" | "<=" | ">" | ">="
               | "&&" | "||"
               ;

Ident           = identifier token ;
String          = string literal token ;
IntLit          = integer literal token ;
FloatLit        = float literal token ;

## 4. Types and Values
1. Builtin types are `int`, `float`, `bool`, `string`, `void`, `any`.
2. User types are struct and enum names.
3. Generic type parameters are allowed only in generic struct/function declarations and are bound by instantiation.
4. Type parameters may be bounded by an interface: `T: InterfaceName`. All instantiations must satisfy the bound.
4. `void` is only valid as a function return type.
5. `any` accepts any value but does not permit implicit conversions to other types.
6. `nil` is not a value. Programs using `nil` are rejected. Use `Option`, `Result`, or `Error`.

## 5. Declarations and Semantics
1. `import "path";` merges declarations from the target file(s) into the program.
2. `struct` defines a product type with named fields.
3. `enum` defines a closed set of variants. Each variant is a value of the enum type.
4. `interface` declares method signatures. `impl` asserts that a struct conforms to an interface.
5. `fn` defines a function with a name, parameters, and return type.
6. `let` at top-level defines a global variable; `const` defines an immutable global.

## 6. Expressions and Statements
1. Expression evaluation is strict and left-to-right.
2. Operators require matching operand types; no implicit numeric conversions exist.
3. `+` concatenates strings when both operands are `string`.
4. `==` and `!=` require operands of the same type.
5. Comparisons `< <= > >=` are allowed only on `int`, `float`, and `string`.
6. `&&` and `||` require `bool` operands.
7. `if` and `while` conditions must be `bool`.

## 7. Type Inference Boundaries
1. `let` or `const` without an explicit type is inferred from the initializer.
2. Function parameters and return types must be explicitly declared.
3. Generic function return types must be fully determined from arguments; otherwise a type error is reported.
4. Generic struct type arguments must be provided explicitly in struct literals.
5. When bounds are present, inferred or explicit type arguments must implement the bound interface.

## 8. Method Calls
1. `value.method(args...)` resolves to a function named `Type_method` where `Type` is the static type of `value`.
2. The receiver is passed as the first argument to the resolved function.
3. If no matching function exists, a type error is reported with suggestions when possible.

## 9. Match (Enum Exhaustiveness)
1. `match` subject expression must be an enum value.
2. A `match` may include guarded arms: `Variant if cond: ...`.
3. Each enum variant must have exactly one unguarded arm; guarded arms do not satisfy exhaustiveness.
4. Guards are evaluated in source order within the matching variant.
5. A `match` statement executes the first arm whose guard is true; otherwise it falls back to the unguarded arm.
6. A `match` expression evaluates to the first arm value whose guard is true; otherwise it evaluates to the unguarded arm value.
7. In a `match` expression, all arm values must have the same type.

## 10. Entry Point Contract
1. Programs must define `fn main(): void { ... }`.
2. `main` must not be generic and must not take parameters.
3. Missing or invalid `main` is a compile error.

## 11. Safety Prelude (Always Available)
Types:
- `struct Error { message: string; }`
- `struct Option[T] { is_some: bool; value: T; }`
- `struct Result[T, E] { is_ok: bool; value: T; err: E; }`

Functions:
- `fn some[T](value: T): Option[T]`
- `fn none[T](fallback: T): Option[T]`
- `fn ok[T, E](value: T, fallback_err: E): Result[T,E]`
- `fn err[T, E](fallback_value: T, err_value: E): Result[T,E]`

Assertions (for tests):
- `fn assert(cond: bool): void`
- `fn assert_msg(cond: bool, msg: string): void`

## 12. Standard Builtins (Always Available)
- `fn print(v: any): void`
- `fn println(v: any): void`
- `fn str(v: any): string`
- `fn len(s: string): int`
- `fn contains(s: string, sub: string): bool`
- `fn starts_with(s: string, prefix: string): bool`
- `fn ends_with(s: string, suffix: string): bool`
- `fn to_upper(s: string): string`
- `fn to_lower(s: string): string`
- `fn trim_space(s: string): string`
- `fn replace(s: string, old: string, new: string): string`
- `fn repeat(s: string, count: int): string`
- `fn parse_int(s: string): Result[int, Error]`
- `fn parse_float(s: string): Result[float, Error]`
- `fn unwrap_or[T](opt: Option[T], fallback: T): T`
- `fn result_or[T, E](res: Result[T,E], fallback: T): T`

## 13. Diagnostics Requirements
1. Lexer and parser errors must include `file:line:col`, source snippet, and caret.
2. Unused local variables and parameters are compile errors; `_` is a discard.
3. Unknown identifiers and members should include a suggestion when a close name exists.
4. Non-`void` functions must return on all control paths.

## 14. Import Resolution and Package Integrity
1. Relative imports (`./`, `../`) are resolved relative to the importing file.
2. Non-relative imports are treated as package aliases resolved under `.bazic/pkg/<alias>`.
3. Absolute paths are disallowed.
4. If `bazic.mod.json` exists, the compiler must verify `bazic.lock.json` and cache integrity before compilation.
5. Import cycles are errors and must report the cycle chain.
6. Local dependency source drift is rejected until `bazc pkg sync` is run.

## 15. Build Determinism
1. `bazc build` must use deterministic Go build flags: `-trimpath` and empty build ID.

## 16. LLVM Backend Status (Non-Normative)
1. LLVM is intended to be feature-complete for the v1.0 surface.
2. Generic functions and structs are monomorphized for LLVM.
3. Interface values are lowered as `{ptr, ptr}`; dynamic dispatch is not yet specified.

## 17. Versioning
- Spec version: v1.0.
- This file is the authoritative spec for v1.0.
