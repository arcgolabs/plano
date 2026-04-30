# Changelog

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
