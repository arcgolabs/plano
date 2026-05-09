# Implementation Status

This document describes the current implementation in this repository relative to [the language draft](../plano_language_definition.md).

Compatibility baseline:

- release: `v0.8.0`
- public API generation: `v1`
- artifact schema: `plano.artifact/v2`

## Implemented

- Lexer for identifiers, strings, ints, floats, durations, sizes, operators, comments, and punctuation
- Parser for:
  - `import`
  - `const`
  - `let`
  - `fn`
  - `return`
  - `break`
  - `continue`
  - `if`
  - `for`
  - form declarations
  - assignments
  - call statements
  - arrays, objects, unary/binary expressions, selectors, indexes, and function calls
  - conditional expressions with `condition ? then : else`
  - membership expressions with `item in list` and `key in map`
- AST model with source positions
- Diagnostics model
- Go workspace layout with separate core, CLI, and example modules
- Separate `lsp` workspace module for downstream language-server integration
- Cobra-based CLI:
  - `plano parse`
  - `plano examples`
  - `plano version`
  - `plano bind`
  - `plano check`
  - `plano hir`
  - `plano compile`
  - `plano validate`
  - `plano diag`
  - embedded sample file listing and display via `plano examples [sample]`
  - `--format` and `--out` output controls
  - `--strict` to fail on any diagnostics
- Taskfile shortcuts:
  - `task fmt`
  - `task test`
  - `task lint`
  - `task bench`
  - `task bench:compiler`
  - `task bench:lsp`
  - `task work:sync`
  - `task parse`
  - `task examples`
  - `task bind`
  - `task check`
  - `task hir`
  - `task compile`
  - `task validate`
  - `task diag`
  - `task sample`
- Schema registration for:
  - forms
  - fields
  - body mode
  - label mode
  - function signatures
  - function result and argument types
  - expr-lang variables and functions exposed by the host
- Compiler support for:
  - explicit bind phase for constants, functions, and declared symbols
  - explicit check phase for expression types, field assignment types, returns, and call signatures
  - explicit typed HIR output for lowering
  - top-level imports
  - glob imports using `**`
  - top-level constants with lazy resolution
  - top-level user-defined functions
  - symbol collection for labeled forms
  - `ref<kind>` validation
  - field type checking
  - nested form validation
  - call-only form bodies with action registry validation
  - script-body execution with lexical scope
  - `let`, local reassignment, `if`, `else if`, single- and dual-variable `for`, `for ... where`, conditional expressions, membership expressions, `break`, and `continue` execution inside script bodies
  - user-defined function execution with typed parameters and returns
  - typed document output
  - serializable `compiler.Artifact` output with explicit schema versioning
  - structured diagnostic suggestions for spelling-based form, field, function, action, and name replacements
  - bounded parse cache for repeated compile requests
  - `expr(...)` and `expr_eval(...)` backed by `github.com/expr-lang/expr`
  - bounded expr-lang program cache for repeated expression compilation
- LSP support module:
  - in-memory workspace document tracking
  - source-based analysis helpers for bytes and strings
  - basic `go.lsp.dev/protocol` server wiring
  - stdio server entrypoint
  - LSP-friendly diagnostics with codes and related information
  - diagnostic-driven quick fixes through compiler-provided suggestions
  - definition lookup
  - hover content generation
  - expr-lang host variable/function hover inside `expr(...)` strings
  - references
  - document symbols
  - folding ranges
  - completion
  - expr-lang host variable/function completion inside `expr(...)` strings
  - prepare rename and rename
  - protocol `textDocument/codeAction` server handling
  - protocol `textDocument/foldingRange` server handling
- Benchmarks:
  - compiler compile-string, compile-artifact, expr-cache, and warm-cache compile-file benchmarks
  - LSP analyze, hover, completion, code action, folding range, expr-lang query, and rename benchmarks
- Example host lowering packages:
  - `examples/builddsl.Register(...)`
  - `examples/builddsl.Lower(...)`
  - `examples/pipelinedsl.Register(...)`
  - `examples/pipelinedsl.Lower(...)`
  - `examples/servicedsl.Register(...)`
  - `examples/servicedsl.Lower(...)`
  - multiple checked-in sample `.plano` files per example DSL
  - `task`, `go.test`, `go.binary`, `pipeline`, `stage`, `stack`, and `service` forms
- Builtin compile-time functions:
  - `env`
  - `join_path`
  - `basename`
  - `dirname`
  - `len`
  - `keys`
  - `values`
  - `range`
  - `get`
  - `slice`
  - `has`
  - `append`
  - `concat`
  - `merge`
- Builtin compile-time globals:
  - `os`
  - `arch`
- Internal error handling:
  - `oops` is used on loader, registration, and lowering error paths
  - `diag.Diagnostics` remains the user-facing DSL error channel

## Intentionally Not Implemented Yet

- Plugin/module packaging API
- Multi-frontend support beyond `.plano`
- Rich error recovery beyond the current structured diagnostic suggestions

## Current Semantic Restrictions

- Identifiers currently allow letters, digits, and `_`
  - Hyphenated identifiers from the draft are not enabled yet.
- `map<T>` currently means a string-keyed map whose values must match `T`
- Call statements are captured structurally in the output document and validated against host-registered actions.
- In script-capable form bodies, assignments prefer declared fields.
  - Non-field local bindings can still be reassigned.
  - Local bindings that collide with field names are rejected.
- Blocks used by `fn`, `if`, and `for` currently accept the same script items as form script bodies
  - This is slightly broader than the stricter draft grammar and is intentional in the current implementation.
- The compiler only accepts these top-level compiled statements:
  - `import`
  - `const`
  - `fn`
  - form declarations
- Other statements may parse successfully, but the compiler will currently report them as unsupported.

## Public APIs

Parser API:

```go
file, diags := plano.ParseFile(fset, "build.plano", src)
```

Binding API:

```go
c := compiler.New(compiler.Options{})
binding, diags := c.BindSource(ctx, "build.plano", src)
```

String binding API:

```go
c := compiler.New(compiler.Options{})
binding, diags := c.BindString(ctx, "build.plano", src)
```

Check API:

```go
c := compiler.New(compiler.Options{})
checks, diags := c.CheckSource(ctx, "build.plano", src)
```

String check API:

```go
c := compiler.New(compiler.Options{})
checks, diags := c.CheckString(ctx, "build.plano", src)
```

Compiler API:

```go
c := compiler.New(compiler.Options{})
doc, diags := c.CompileSource(ctx, "build.plano", src)
```

String compiler API:

```go
c := compiler.New(compiler.Options{})
doc, diags := c.CompileString(ctx, "build.plano", src)
```

Detailed compiler API:

```go
c := compiler.New(compiler.Options{})
result := c.CompileSourceDetailed(ctx, "build.plano", src)
_ = result.Binding
_ = result.Checks
_ = result.HIR
```

Artifact API:

```go
c := compiler.New(compiler.Options{})
artifact, err := c.CompileSourceArtifact(ctx, "build.plano", src)
data, err := artifact.MarshalJSON()
```

Build lowering API:

```go
c := compiler.New(compiler.Options{})
_ = builddsl.Register(c) // import from github.com/arcgolabs/plano/examples/builddsl
doc, diags := c.CompileSource(ctx, "build.plano", src)
result := c.CompileSourceDetailed(ctx, "build.plano", src)
project, err := builddsl.Lower(result.HIR)
```

LSP server API:

```go
base := compiler.New(compiler.Options{})
workspace := lsp.NewWorkspace(lsp.Options{Compiler: base})
server := lsp.NewServer(lsp.ServerOptions{Workspace: workspace})
err := lsp.ServeStdio(ctx, lsp.ServerOptions{Workspace: workspace})
```

CLI:

```bash
go run ./cmd/plano examples
go run ./cmd/plano parse ./build.plano
go run ./cmd/plano bind ./build.plano
go run ./cmd/plano check ./build.plano
go run ./cmd/plano hir ./build.plano
go run ./cmd/plano compile ./build.plano
go run ./cmd/plano validate ./build.plano
go run ./cmd/plano diag ./build.plano
```

## Test Coverage

Current automated tests cover:

- AST parsing of representative DSL input
- parse diagnostics on malformed source
- constant resolution and expression evaluation
- symbol/reference resolution
- import loading
- glob import expansion
- schema-based field validation
- static typechecking for expressions, returns, fields, and registered call signatures
- script-body execution and user-defined functions
- action validation for call statements
- typed HIR generation
- example builddsl, pipelinedsl, and servicedsl lowering
- bundled sample `.plano` scripts for each example DSL
- smoke tests that lower every checked-in sample file under each example directory

Run with:

```bash
go test ./... ./cmd/plano/... ./examples/builddsl/... ./examples/pipelinedsl/... ./examples/servicedsl/... ./lsp/...
```

## Near-Term Direction

The current implementation is already useful as an embeddable compiler core. The next likely areas of work are:

- richer collection transforms and data-update operations in script bodies
- stronger diagnostics with related spans, better import-cycle reporting, and cleaner host validation feedback
- continued HIR stabilization so example DSL lowering patterns can be externalized later without reworking core phases
- more example DSL scenarios that exercise imports, references, and larger script-heavy documents

## Workspace Layout

The repository now uses `go.work` to stitch together a few focused modules:

- root module: compiler core and public language packages
- `cmd/plano`: CLI distribution module
- `lsp`: LSP helper module
- `examples/builddsl`: build-oriented example DSL module
- `examples/pipelinedsl`: pipeline-oriented example DSL module
- `examples/servicedsl`: service-topology example DSL module

This keeps the compiler core free of CLI/example dependency drag and gives future modules such as `lsp` or plugin adapters a clear place to live.

Sibling workspace modules are intentionally resolved by `go.work` rather than being duplicated as explicit local module requirements inside each child `go.mod`.
