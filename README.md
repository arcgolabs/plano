# plano

`plano` is an embeddable, schema-driven DSL runtime written in Go.

Current baseline:

- release: `v0.1.0`
- public API generation: `v1`
- artifact schema: `plano.artifact/v1`

This repository currently contains a first usable implementation with:

- hand-written lexer and parser
- AST and diagnostics
- a Go workspace (`go.work`) with separate core, CLI, and example modules
- schema registration for forms and functions
- a Cobra-based CLI under `cmd/plano`
- a public bind API for declaration and symbol collection
- a public check API for static type analysis
- a public HIR phase for typed compiler-internal lowering input
- a compiler that produces a typed document
- script-body execution with lexical scope and user-defined functions
- script control flow with `else if`, `break`, and `continue`
- bundled example host DSLs under `examples/`
- validated action registry for call statements
- glob imports via `**`
- a `Taskfile.yml` for common local commands
- unit tests for parsing, compilation, imports, script execution, and lowering

## Packages

- `cmd/plano`: CLI for parsing, compiling, and lowering `.plano` files
- `frontend/plano`: `ParseFile` API for `.plano` source to AST
- `compiler`: structured compile API from source bytes, strings, or files to typed documents
- `lsp`: workspace analysis plus a basic `go.lsp.dev/protocol` LSP server with hover, definition, and diagnostics
- `schema`: form specs, field specs, types, refs, and builtin scalar types
- `ast`: parser output nodes
- `diag`: diagnostics model

Examples:

- `examples/builddsl`: build graph lowering
- `examples/pipelinedsl`: CI pipeline lowering
- `examples/servicedsl`: service topology lowering

Each bundled example now ships with multiple `.plano` scripts so the repository exercises not only the host registration/lowering code, but also representative language features such as control flow, collection builtins, and derived field expressions.

The implementation also uses:

- `collectionx` for ordered compiler outputs, object values, and host-side IR structures
- `mo` for optional values in lowered IR
- `lo` for concise lowering transforms
- `oops` for internal error wrapping on loader and lowering boundaries

## Quick Example

```go
package main

import (
    "context"

    "github.com/arcgolabs/plano/compiler"
    "github.com/arcgolabs/plano/schema"
)

func main() {
    c := compiler.New(compiler.Options{})

    _ = c.RegisterForm(schema.FormSpec{
        Name:      "workspace",
        LabelKind: schema.LabelNone,
        BodyMode:  schema.BodyFieldOnly,
        Fields: schema.Fields(
            schema.FieldSpec{
                Name:     "name",
                Type:     schema.TypeString,
                Required: true,
            },
        ),
    })

    _, _ = c.CompileSource(context.Background(), "build.plano", []byte(`
workspace {
  name = "demo"
}
`))
}
```

You can also inspect the declaration-binding phase directly:

```go
binding, diags := c.BindSource(context.Background(), "build.plano", src)
_ = binding
_ = diags
```

The compiler also exposes string helpers when you already have in-memory source text:

```go
result := c.CompileStringDetailed(ctx, "build.plano", src)
_ = result.Document
```

And the static typecheck phase:

```go
checks, diags := c.CheckSource(context.Background(), "build.plano", src)
_ = checks
_ = diags
```

And the typed HIR phase:

```go
result := c.CompileSourceDetailed(ctx, "build.plano", src)
_ = result.HIR
```

For editor integrations, the `lsp` module can either analyze in-memory documents directly or expose a basic LSP server:

```go
server := lsp.NewServer(lsp.ServerOptions{
    Compiler: configuredCompiler,
})
_ = lsp.ServeStdio(context.Background(), lsp.ServerOptions{
    Compiler: configuredCompiler,
})
```

## Docs

- Language draft: [plano_language_definition.md](./plano_language_definition.md)
- Changelog: [CHANGELOG.md](./CHANGELOG.md)
- Implementation status: [docs/implementation_status.md](./docs/implementation_status.md)
- Public API policy: [docs/public_api.md](./docs/public_api.md)
- Artifact schema: [docs/artifact_schema.md](./docs/artifact_schema.md)

## CLI

Build and run:

```bash
go run ./cmd/plano examples
go run ./cmd/plano version
go run ./cmd/plano parse ./build.plano
go run ./cmd/plano bind --example builddsl ./build.plano
go run ./cmd/plano check --example builddsl ./build.plano
go run ./cmd/plano hir --example builddsl ./build.plano
go run ./cmd/plano compile --example builddsl ./build.plano
go run ./cmd/plano lower --example builddsl ./build.plano
go run ./cmd/plano validate --example builddsl ./build.plano
go run ./cmd/plano diag --example builddsl ./build.plano
go run ./cmd/plano lower --example builddsl --format yaml --out ./project.yaml ./build.plano
```

`parse` prints AST JSON.
`bind` prints the declaration and symbol binding result JSON.
`check` prints the binding plus static typecheck result JSON.
`hir` prints the typed HIR JSON.
`compile` prints the typed document JSON.
`lower` compiles with a registered example host DSL and prints the lowered IR JSON.
`validate` checks whether the file compiles successfully.
`diag` prints diagnostics without failing the command on warnings.
`examples` lists each bundled DSL together with every checked-in sample script for that example.
`version` prints the release version, public API generation, and artifact schema generation.

Output controls:

- `--format json|yaml` for `parse`, `compile`, and `lower`
- `--format text|json|yaml` for `validate` and `diag`
- `--out <path>` to write command output to a file instead of stdout
- `--strict` on `compile`, `lower`, and `validate` to fail on any diagnostics, not only errors

## Taskfile

The repository also ships a small `Taskfile.yml` for common local workflows:

```bash
task fmt
task test
task lint
task bench
task bench:compiler
task bench:lsp
task work:sync
task examples
task parse FILE=./build.plano FORMAT=yaml
task bind FILE=./build.plano EXAMPLE=builddsl FORMAT=yaml
task check FILE=./build.plano EXAMPLE=builddsl FORMAT=yaml
task hir FILE=./build.plano EXAMPLE=builddsl FORMAT=yaml
task lower FILE=./build.plano EXAMPLE=builddsl FORMAT=yaml OUT=./project.yaml
task example:builddsl FORMAT=yaml
task example:pipelinedsl FORMAT=yaml
task example:servicedsl FORMAT=yaml
```

## Repo Shape

The repository now runs as a small Go workspace rather than a single module.

- Root module `github.com/arcgolabs/plano`: compiler core, AST, diagnostics, schema, and frontend packages
- `cmd/plano`: standalone CLI module
- `lsp`: standalone LSP helper module
- `examples/builddsl`: example build DSL module
- `examples/pipelinedsl`: example pipeline DSL module
- `examples/servicedsl`: example service DSL module

That split gives us cleaner boundaries:

- core can evolve without dragging CLI/example dependencies into every consumer build
- example DSLs are now visibly host-side modules instead of looking like core packages
- future modules such as `lsp` or plugin/runtime adapters can be added without reshaping the core again

Workspace note:

- sibling workspace modules are resolved through `go.work`
- child `go.mod` files only declare external dependencies, not other local workspace modules

## Current Scope

The implementation is still narrower than the full language draft, but the main compiler path is now usable:

- imports
- glob imports such as `import "tasks/**/*.plano"`
- top-level `const`
- top-level user-defined `fn`
- builtins such as `len`, `keys`, `values`, `range`, `get`, `slice`, `has`, `append`, `concat`, and `merge`
- static typechecking for expressions, fields, returns, and registered function/action signatures
- validated call statements through host-registered actions
- typed HIR output for stable lowering
- form declarations
- script-body execution with `let`, local reassignment, `if`, `else if`, single- and dual-variable `for`, `break`, and `continue`
- field assignments, nested forms, and call statements
- expression evaluation with registered and user-defined functions
- lowering from HIR to sample IRs through `examples/builddsl`, `examples/pipelinedsl`, and `examples/servicedsl`

Plugin packaging and richer module/runtime integration are still pending.

## Near-Term Direction

The repository is now past the "parser prototype" stage and behaves more like a real compiler core. The next useful iterations are mostly semantic and tooling work:

- keep extending collection and script ergonomics without collapsing host DSL boundaries into core
- tighten diagnostics, especially related spans and richer import/reference errors
- stabilize the HIR and example-lowering contracts before introducing a formal plugin packaging API
- add more real-world example DSL flows so language changes are exercised against multiple host shapes
