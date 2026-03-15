# Migration Tooling (Draft)

This document defines how Bazic will manage breaking changes, deprecations, and automated migrations.

## Goals
- Provide deterministic, compiler-assisted migrations.
- Detect breaking changes early via CI gates.
- Keep v1.x code compiling without changes.

## Proposed tools
### 1) `bazic migrate`
- Runs a set of migration rules against a project.
- Applies safe transforms or emits a patch set for review.

### 2) `bazic check --compat`
- Verifies compatibility against a declared target version (e.g. `v1.0`).
- Fails CI if deprecated features are used after their removal date.
 - Implemented baseline check for spec stability (LANGUAGE.md not Draft).

## Migration lifecycle
1. Deprecation introduced (minor release)
2. Warning emitted in compiler
3. Migration tool provides fix
4. Feature removed in next major release

## Required artifacts
- `MIGRATIONS.md` entry per breaking change
- Migration rule definitions (JSON or Bazic AST rules)
- Automated tests for each rule
