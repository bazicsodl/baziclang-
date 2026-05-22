# Bazic Roadmap

This roadmap is the execution plan for taking Bazic from its current compiler prototype state to a production-grade public release.

## Operating Rules

- The Go backend is the release backend for Bazic v1.
- The LLVM backend remains experimental until it reaches backend parity and runtime stability.
- Compiler architecture work takes priority over new language surface area.
- Web/UI scaffolding, generators, and app templates are not allowed to block compiler-core milestones.
- Every milestone must leave the repository in a releasable state.

## Phase 0: Product Lock

Goal: remove ambiguity about what Bazic v1 is.

Key tasks:
- Define Bazic v1 goals, non-goals, and supported workloads.
- Freeze release backend policy.
- Freeze the initial supported platform matrix.
- Define what must be stable by 1.0 and what stays experimental.

Expected outputs:
- `RELEASE_SCOPE.md`
- Updated top-level docs that match actual release posture

Exit gate:
- The team can answer "what is Bazic 1.0?" in one page without contradiction.

## Phase 1: Repository Cleanup

Goal: make the repo maintainable enough for sustained compiler work.

Key tasks:
- Separate compiler, runtime, tools, examples, docs, and experimental areas.
- Archive or isolate legacy Python prototypes and old snapshots.
- Remove checked-in binaries, temp IR, and generated artifacts from tracked source.
- Define clean build-output conventions.

Expected outputs:
- Clean top-level layout
- Tightened `.gitignore`
- Reduced accidental churn in the worktree

Exit gate:
- A contributor can find frontend, backend, runtime, tests, and docs immediately.

## Phase 2: Spec And Docs Alignment

Goal: make documentation truthful.

Key tasks:
- Align grammar docs with the actual parser.
- Remove stale syntax examples.
- Document backend status honestly.
- Mark authoritative vs draft documents clearly.

Expected outputs:
- Updated `README.md`
- Updated `LANGUAGE.md`
- Updated `LANGUAGE_SPEC.md`

Exit gate:
- Every syntax example in public docs parses and checks today.

## Phase 3: Span-Aware Frontend

Goal: give the compiler real source locations end to end.

Key tasks:
- Add spans to tokens.
- Add spans to AST nodes.
- Preserve file identity through import merging.
- Replace regex-parsed location extraction with structured diagnostics.

Expected outputs:
- Span-aware lexer/parser/AST
- Structured compiler diagnostics

Exit gate:
- Lexer, parser, and semantic errors all report exact source ranges without string parsing.

## Phase 4: Structured Type System

Goal: remove the stringly-typed type model.

Key tasks:
- Replace `ast.Type string` with structured type nodes.
- Model builtins, named types, generic instances, type parameters, and invalid/error types explicitly.
- Remove ad hoc parsing of formatted types like `Result[int,Error]`.

Expected outputs:
- New type representation
- Refactored semantic logic

Exit gate:
- No compiler pass depends on string formatting to understand a type.

## Phase 5: Multi-Pass Semantic Analysis

Goal: make the frontend scalable and testable.

Key tasks:
- Split semantic analysis into symbol collection, resolution, type checking, impl validation, and control-flow analysis.
- Stop mutating AST during semantic checking.
- Centralize intrinsic/builtin registration.

Expected outputs:
- Explicit semantic passes
- Cleaner compiler pipeline

Exit gate:
- Method resolution, match typing, and builtin handling no longer depend on mutating user AST nodes.

## Phase 6: Typed HIR

Goal: introduce a stable compiler-internal representation after parsing.

Key tasks:
- Lower AST to typed HIR.
- Resolve symbols and desugar methods/prelude/import effects before backend work.
- Keep HIR source-aware for diagnostics and tooling.

Expected outputs:
- `internal/hir`
- HIR lowering tests

Exit gate:
- Tooling and backend preparation can consume HIR without redoing semantic work.

## Phase 7: MIR

Goal: stop compiling directly from syntax-level trees.

Key tasks:
- Define MIR with explicit blocks, temporaries, branches, calls, and intrinsics.
- Lower match/control-flow constructs into explicit CFG form.
- Add MIR validation.

Expected outputs:
- `internal/mir`
- MIR verifier
- Golden MIR tests

Exit gate:
- Every compilable sample lowers to valid MIR deterministically.

## Phase 8: MIR-Based Go Backend

Goal: keep the bootstrap backend while making it architecturally clean.

Key tasks:
- Port Go codegen from AST to MIR.
- Consolidate runtime/builtin signatures behind a shared intrinsic registry.
- Remove duplicated lowering logic.

Expected outputs:
- MIR-driven Go backend

Exit gate:
- Existing conformance suite passes on MIR -> Go.

## Phase 9: MIR-Based LLVM Backend

Goal: make native compilation credible.

Key tasks:
- Port LLVM lowering from AST/text-emission patterns to MIR.
- Define stable ABI and type layout rules.
- Keep runtime calls explicit and typed.
- Add parity testing against the Go backend.

Expected outputs:
- MIR-driven LLVM backend
- Backend parity suite

Exit gate:
- Core conformance tests pass on both Go and LLVM.

## Phase 10: Runtime Model

Goal: define Bazic's actual safety and execution model.

Key tasks:
- Lock v1 memory strategy.
- Define string, collection, file, and process runtime contracts.
- Define safe vs unsafe boundaries for native interop.
- Reduce duplicated runtime behavior across backends.

Expected outputs:
- `RUNTIME_MODEL.md`
- Tight runtime ABI contracts

Exit gate:
- Every intrinsic and builtin has a documented runtime contract.

## Phase 11: Standard Library Hardening

Goal: ship a trustworthy core stdlib.

Key tasks:
- Define a stable core tier: `core`, `io`, `fs`, `time`, `json`, `http`, `os`, `path`, `test`.
- Mark `db`, `auth`, `desktop`, `ui`, `web`, and generators as experimental until hardened.
- Expand tests around stable stdlib APIs.

Expected outputs:
- Tiered stdlib surface
- Stability labels in docs

Exit gate:
- Release stdlib is documented, tested, and backend-compatible.

## Phase 12: Tooling Unification

Goal: stop re-implementing compiler logic in side tools.

Key tasks:
- Make formatter, linter, test runner, and LSP use shared compiler services.
- Replace regex-based rename/navigation with symbol-aware operations.
- Make diagnostics consistent across CLI and editor tooling.

Expected outputs:
- Shared compiler service layer
- More reliable LSP behavior

Exit gate:
- Rename, hover, and diagnostics are symbol-aware rather than raw text-driven.

## Phase 13: Modules And Packages

Goal: support larger Bazic codebases cleanly.

Key tasks:
- Add package declarations and visibility.
- Add explicit exports.
- Strengthen manifest, lockfile, versioning, and workspace behavior.
- Remove global declaration merging as the long-term model.

Expected outputs:
- Real module/package system

Exit gate:
- Multi-package apps compile without global namespace hacks.

## Phase 14: Release Quality Gates

Goal: make regressions expensive.

Key tasks:
- Expand golden tests across parser, sema, HIR, MIR, and both backends.
- Add fuzz targets for lexer/parser/typechecker.
- Add benchmark gates for compiler and runtime hotspots.
- Add backend parity checks in CI.

Expected outputs:
- Strong CI release gates

Exit gate:
- Every release candidate passes correctness, parity, and performance checks.

## Phase 15: Alpha

Goal: stabilize with real users.

Key tasks:
- Publish nightly builds.
- Select reference apps that exercise the supported core.
- Triage compiler crashes, diagnostic failures, and package workflow issues.
- Freeze non-critical feature work.

Expected outputs:
- Alpha feedback log
- Reduced crash rate

Exit gate:
- No open crashers or major correctness bugs in the supported core.

## Phase 16: Beta

Goal: verify release readiness.

Key tasks:
- Finalize installers and package flow.
- Finalize public docs and migration notes.
- Run platform smoke tests.
- Prepare release signing and distribution process.

Expected outputs:
- Release candidates
- Finalized installation story

Exit gate:
- Supported OS targets install and run the official quickstart successfully.

## Phase 17: Bazic 1.0

Goal: go live.

Key tasks:
- Tag and publish Bazic 1.0.
- Publish release notes and compatibility policy.
- Publish checksums and installers.
- Open the patch-release cadence.

Expected outputs:
- Public Bazic 1.0 release

Exit gate:
- Users can install Bazic, create a project, build it, test it, and ship it using the published docs alone.

## Immediate Execution Queue

1. Finish Phase 0 documentation and align the top-level docs.
2. Start Phase 1 repository cleanup with a proposed target layout.
3. Start Phase 3 by adding spans to tokens and AST nodes.
4. Replace string-parsed diagnostics with structured location reporting.
5. Design the structured type model before more frontend feature work.
