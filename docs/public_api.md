# Public API

As of `plano` `v0.3.0`, the repository treats the following packages as the supported public surface:

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

- Module release: `v0.3.0`
- Public API generation: `v1`
- Artifact schema generation: `plano.artifact/v1`

Compatibility rules:

- Exported identifiers in the packages above are treated as stable within the `v0.x` baseline unless explicitly documented otherwise.
- `internal/*`, `cmd/*`, and `examples/*` are not compatibility targets.
- `compiler.Artifact` is the only supported serialized compiler artifact shape.
- `compiler.Result`, `compiler.Binding`, `compiler.CheckInfo`, `compiler.HIR`, and `lsp.Snapshot` remain in-memory APIs and are not persistence contracts by themselves.

Extension guidance:

- Register forms and functions through `schema.FormSpec`, `schema.FunctionSpec`, and `compiler.ActionSpec`.
- Register expr-lang variables and functions through `compiler.RegisterExprVar`, `compiler.RegisterExprFunc`, and `compiler.RegisterExprFunction`.
- Prefer `collectionx`-backed helpers such as `schema.Fields`, `schema.NestedForms`, and `schema.Types` when constructing specs.
- Persist compiler outputs through `compiler.Artifact`, not raw `compiler.Result`.
