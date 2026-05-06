# Changelog

## v0.6.0

Compiler diagnostics, CLI boundary, and syntax iteration release.

Highlights:

- added structured diagnostic suggestions to `diag.Diagnostic`
- moved spelling-based quick-fix candidate generation from LSP into the compiler
- added compiler suggestions for unknown fields and invalid nested form names
- added `for ... where` filtered loop syntax for script-capable bodies and functions
- added conditional expression syntax with `condition ? then : else`
- added conditional-expression samples for the CLI and bundled example DSL modules
- added `diagnostics[].suggestions` to compiler artifacts
- updated `lsp.Snapshot.CodeActions` to consume diagnostic suggestions instead of parsing diagnostic messages
- decoupled `cmd/plano` from documentation example packages and replaced example-backed lowering with embedded sample display
- bumped artifact schema generation to `plano.artifact/v2` while keeping v1 artifacts readable
- kept public API generation at `v1`

## v0.5.0

LSP tooling and dependency refresh release.

Highlights:

- added `lsp.Snapshot.CodeActions` for diagnostic-driven quick fixes
- added `textDocument/codeAction` support to the protocol server
- added spelling-based replacement quick fixes for unknown forms, functions, actions, and unresolved names
- added `lsp.Snapshot.FoldingRanges` and protocol `textDocument/foldingRange` support
- optimized LSP span-to-position conversion and cached folding range queries
- updated arcgolabs `collectionx` dependencies to their latest valid versions
- added LSP code action and folding range tests plus benchmark coverage
- kept public API generation at `v1`
- kept artifact schema at `plano.artifact/v1`

## v0.4.0

Expression tooling and cache release.

Highlights:

- added bounded expr-lang program caching through `compiler.Options.ExprCacheEntries`
- added expr-lang host variable/function completion and hover support in the `lsp` module
- added compiler benchmarks for warm, cold, and disabled expr cache scenarios
- added LSP benchmarks for expr-lang completion and hover
- optimized repeated expr cache-key construction by caching registered function signatures
- kept public API generation at `v1`
- kept artifact schema at `plano.artifact/v1`

## v0.3.0

Expr integration release.

Highlights:

- added `github.com/expr-lang/expr` as an opt-in expression evaluator
- added `expr(...)` and `expr_eval(...)` builtins for dynamic expression strings
- added host APIs for expr-lang variables and functions through `RegisterExprVar`, `RegisterExprFunc`, and `RegisterExprFunction`
- kept public API generation at `v1`
- kept artifact schema at `plano.artifact/v1`

## v0.2.0

Performance and benchmark expansion release.

Highlights:

- added larger benchmark scenarios for parser, compiler, and LSP query paths
- optimized LSP reference queries with snapshot-level lazy caching
- optimized repeated LSP document symbol queries with snapshot-level result reuse
- kept public API generation at `v1`
- kept artifact schema at `plano.artifact/v1`

## v0.1.0

Initial baseline release.

Highlights:

- stable public package baseline for `frontend/plano`, `compiler`, `schema`, `diag`, and `lsp`
- explicit compiler phases: bind, check, HIR, compile
- serializable `compiler.Artifact` contract with schema version `plano.artifact/v1`
- bounded compiler parse cache and LSP workspace analysis cache
- richer diagnostics with codes and related information
- LSP support for diagnostics, hover, definition, references, document symbols, completion, prepare rename, and rename
- bundled example DSLs for build, pipeline, and service topologies
- compiler and LSP benchmark coverage plus task shortcuts
