# Bazic Full-Stack Plan

This is the concrete plan to make Bazic a stable, full-stack platform (backend + frontend) with first-class auth, models, database access, routing, and interoperability.

## Core Pillars
1. **Language Stability**
   - Spec lock for v1.0, compatibility matrix, semantic versioning.
   - Standard library stability rules (no breaking changes across minor releases).
   - Conformance test suite expanded for web, DB, crypto, http server, and serialization.

2. **Backend Framework (Std + Runtime)**
   - HTTP server with routing, params, cookies, headers, and query helpers.
   - JSON helpers (validate, pretty, minify) and structured parsing/encoding.
   - Middleware conventions, request context, and error handling patterns.
   - Sessions and auth helpers in stdlib.

3. **Database + Models**
   - SQLite (default), PostgreSQL, MySQL, SQL Server, and Mongo (drivers).
   - Schema migrations toolchain (migration files, apply/rollback, status).
   - ORM-like helpers (query builder or generated models).

4. **Security & Crypto**
   - Bcrypt + Argon2 + PBKDF2
   - Secure random, JWT, HMAC, constant-time compare
   - Cookie security helpers (httpOnly, sameSite, secure, max-age)

5. **Frontend Runtime**
   - Bazic UI (WASM) + hydration support
   - Unified API client helpers
   - Build tooling that outputs static assets + WASM + SSR option

6. **Interoperability**
   - HTTP + gRPC + OpenAPI generation
   - FFI bridges for Go, JS, Java
   - Package manager compatibility and artifact publishing

7. **Dev Experience**
   - CLI scaffolding for full-stack apps
   - Integrated testing (unit + http integration)
   - Formatter + linter as hard gates

## Milestones
### M1: Server + Auth Foundation (Now)
- Go backend: HTTP server with routing conventions
- Cookie, header, query, param parsing helpers
- Bcrypt hash + verify in stdlib
- Example webstack app

### M2: Sessions + Auth + JSON parsing
- Secure session helpers (DB + in-memory)
- JSON parse to structs and maps
- Request validation patterns

### M3: DB + Models
- Model schema format
- Migrations tool
- Model codegen

### M4: Frontend + SSR
- Bazic UI SSR and hydration hooks
- API client for Bazic UI + external backends

### M5: Interop
- Go/Java/JS bindings
- OpenAPI generator

### M6: v1.0 Stability
- Freeze core API
- Expand conformance suites
- Release tooling

## Immediate Implementation (Completed in this change)
- `http_serve_app` server routing and request/response structs
- Routing conventions by function name
- Cookie/header/query/param helpers
- Bcrypt hash/verify for Go backend
- LLVM stubs for clear unsupported features
- Example app `examples/apps/webstack`

## Next Implementation Steps (Queued)
1. Add JSON parsing into `std/json` (decode to `map[string]string` and basic structs).
2. Add session store: in-memory and DB-backed.
3. Add auth helper package (password + token + cookie policies).
4. Add migrations tool for `std/db`.
5. Add OpenAPI generator for Bazic server apps.
