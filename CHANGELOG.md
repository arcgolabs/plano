# Changelog

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
