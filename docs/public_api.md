# Public API

As of `plano` `v0.6.0`, the repository treats the following packages as the supported public surface:

- `github.com/arcgolabs/plano`
  - module and compatibility metadata
- `github.com/arcgolabs/plano/frontend/plano`
  - parsing `.plano` source into AST plus diagnostics
- `github.com/arcgolabs/plano/compiler`
  - schema registration, bind/check/compile APIs, typed documents, HIR, and serializable artifacts
- `github.com/arcgolabs/plano/schema`
  - host-side form, field, function, and type registration model
- `github.com/arcgolabs/plano/diag`
  - compiler/frontend diagnostic model
- `github.com/arcgolabs/plano/lsp`
  - workspace analysis helpers and protocol-facing LSP server

Compatibility baseline:

- Module release: `v0.6.0`
- Public API generation: `v1`
- Artifact schema generation: `plano.artifact/v2`

Compatibility rules:

- Exported identifiers in the packages above are treated as stable within the `v0.x` baseline unless explicitly documented otherwise.
- `internal/*`, `cmd/*`, and `examples/*` are not compatibility targets.
- `compiler.Artifact` is the only supported serialized compiler artifact shape.
- `compiler.Result`, `compiler.Binding`, `compiler.CheckInfo`, `compiler.HIR`, and `lsp.Snapshot` remain in-memory APIs and are not persistence contracts by themselves.

Extension guidance:

- Register forms and functions through `schema.FormSpec`, `schema.FunctionSpec`, and `compiler.ActionSpec`.
- Register expr-lang variables and functions through `compiler.RegisterExprVar`, `compiler.RegisterExprFunc`, and `compiler.RegisterExprFunction`.
- Tune repeated expr-lang compilation through `compiler.Options.ExprCacheEntries`; `0` uses the default bounded cache and `-1` disables it.
- Use `diag.Diagnostic.Suggestions` when presenting compiler-provided quick fixes.
- Use `lsp.Snapshot.FoldingRanges` for editor folding support.
- Use `lsp.Snapshot.CodeActions` for diagnostic-driven editor quick fixes.
- Prefer `collectionx`-backed helpers such as `schema.Fields`, `schema.NestedForms`, and `schema.Types` when constructing specs.
- Persist compiler outputs through `compiler.Artifact`, not raw `compiler.Result`.
