# plano

`plano` is an embeddable, schema-driven DSL runtime written in Go.

This repository currently contains a first usable implementation with:

- hand-written lexer and parser
- AST and diagnostics
- schema registration for forms and functions
- a Cobra-based CLI under `cmd/plano`
- a compiler that produces a typed document
- script-body execution with lexical scope and user-defined functions
- an example host DSL under `examples/builddsl`
- validated action registry for call statements
- glob imports via `**`
- a `Taskfile.yml` for common local commands
- unit tests for parsing, compilation, imports, script execution, and lowering

## Packages

- `cmd/plano`: CLI for parsing, compiling, and lowering `.plano` files
- `frontend/plano`: `ParseFile` API for `.plano` source to AST
- `compiler`: structured compile API from source/file to typed document
- `schema`: form specs, field specs, types, refs, and builtin scalar types
- `ast`: parser output nodes
- `diag`: diagnostics model

Examples:

- `examples/builddsl`: sample host DSL registration and lowering to build IR

The implementation also uses:

- `collectionx` for ordered host-side IR structures
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
        Fields: map[string]schema.FieldSpec{
            "name": {
                Name:     "name",
                Type:     schema.TypeString,
                Required: true,
            },
        },
    })

    _, _ = c.CompileSource(context.Background(), "build.plano", []byte(`
workspace {
  name = "demo"
}
`))
}
```

## Docs

- Language draft: [plano_language_definition.md](./plano_language_definition.md)
- Implementation status: [docs/implementation_status.md](./docs/implementation_status.md)

## CLI

Build and run:

```bash
go run ./cmd/plano parse ./build.plano
go run ./cmd/plano compile --example builddsl ./build.plano
go run ./cmd/plano lower --example builddsl ./build.plano
go run ./cmd/plano validate --example builddsl ./build.plano
go run ./cmd/plano diag --example builddsl ./build.plano
go run ./cmd/plano lower --example builddsl --format yaml --out ./project.yaml ./build.plano
```

`parse` prints AST JSON.
`compile` prints the typed document JSON.
`lower` compiles with a registered example host DSL and prints the lowered IR JSON.
`validate` checks whether the file compiles successfully.
`diag` prints diagnostics without failing the command on warnings.

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
task parse FILE=./build.plano FORMAT=yaml
task lower FILE=./build.plano EXAMPLE=builddsl FORMAT=yaml OUT=./project.yaml
```

## Repo Shape

This repository intentionally remains a single Go module for now.

- The core compiler APIs are still moving quickly.
- Splitting into many `go.mod` files now would increase local development and testing overhead.
- The CLI and examples already give us the separation we need without multi-module versioning friction.

## Current Scope

The implementation is still narrower than the full language draft, but the main compiler path is now usable:

- imports
- glob imports such as `import "tasks/**/*.plano"`
- top-level `const`
- top-level user-defined `fn`
- validated call statements through host-registered actions
- form declarations
- script-body execution with `let`, `if`, and `for`
- field assignments, nested forms, and call statements
- expression evaluation with registered and user-defined functions
- lowering from `Document` to a sample build IR through `examples/builddsl`

Plugin packaging and richer module/runtime integration are still pending.
