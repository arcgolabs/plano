# Implementation Status

This document describes the current implementation in this repository relative to [the language draft](../plano_language_definition.md).

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
- AST model with source positions
- Diagnostics model
- Cobra-based CLI:
  - `plano parse`
  - `plano examples`
  - `plano bind --example builddsl`
  - `plano check --example builddsl`
  - `plano hir --example builddsl`
  - `plano compile --example builddsl`
  - `plano lower --example builddsl`
  - `plano validate --example builddsl`
  - `plano diag --example builddsl`
  - `--format` and `--out` output controls
  - `--strict` to fail on any diagnostics
- Taskfile shortcuts:
  - `task fmt`
  - `task test`
  - `task lint`
  - `task parse`
  - `task examples`
  - `task bind`
  - `task check`
  - `task hir`
  - `task compile`
  - `task lower`
  - `task validate`
  - `task diag`
- Schema registration for:
  - forms
  - fields
  - body mode
  - label mode
  - function signatures
  - function result and argument types
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
  - `let`, local reassignment, `if`, `else if`, single- and dual-variable `for`, `break`, and `continue` execution inside script bodies
  - user-defined function execution with typed parameters and returns
  - typed document output
- Example host lowering packages:
  - `examples/builddsl.Register(...)`
  - `examples/builddsl.Lower(...)`
  - `examples/pipelinedsl.Register(...)`
  - `examples/pipelinedsl.Lower(...)`
  - `examples/servicedsl.Register(...)`
  - `examples/servicedsl.Lower(...)`
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
- Rich error recovery and diagnostic suggestions

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

Check API:

```go
c := compiler.New(compiler.Options{})
checks, diags := c.CheckSource(ctx, "build.plano", src)
```

Compiler API:

```go
c := compiler.New(compiler.Options{})
doc, diags := c.CompileSource(ctx, "build.plano", src)
```

Detailed compiler API:

```go
c := compiler.New(compiler.Options{})
result := c.CompileSourceDetailed(ctx, "build.plano", src)
_ = result.Binding
_ = result.Checks
_ = result.HIR
```

Build lowering API:

```go
c := compiler.New(compiler.Options{})
_ = builddsl.Register(c) // import from github.com/arcgolabs/plano/examples/builddsl
doc, diags := c.CompileSource(ctx, "build.plano", src)
result := c.CompileSourceDetailed(ctx, "build.plano", src)
project, err := builddsl.Lower(result.HIR)
```

CLI:

```bash
go run ./cmd/plano examples
go run ./cmd/plano parse ./build.plano
go run ./cmd/plano bind --example builddsl ./build.plano
go run ./cmd/plano check --example builddsl ./build.plano
go run ./cmd/plano hir --example builddsl ./build.plano
go run ./cmd/plano compile --example builddsl ./build.plano
go run ./cmd/plano lower --example builddsl ./build.plano
go run ./cmd/plano validate --example builddsl ./build.plano
go run ./cmd/plano diag --example builddsl ./build.plano
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

Run with:

```bash
go test ./...
```
