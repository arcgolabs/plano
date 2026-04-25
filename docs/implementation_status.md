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
- Compiler support for:
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
  - `let`, `if`, and `for` execution inside script bodies
  - user-defined function execution with typed parameters and returns
  - typed document output
- Example host lowering package:
  - `examples/builddsl.Register(...)`
  - `examples/builddsl.Lower(...)`
  - `task`, `go.test`, and `go.binary` forms
- Builtin compile-time functions:
  - `env`
  - `join_path`
  - `basename`
  - `dirname`
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

Compiler API:

```go
c := compiler.New(compiler.Options{})
doc, diags := c.CompileSource(ctx, "build.plano", src)
```

Build lowering API:

```go
c := compiler.New(compiler.Options{})
_ = builddsl.Register(c) // import from github.com/arcgolabs/plano/examples/builddsl
doc, diags := c.CompileSource(ctx, "build.plano", src)
project, err := builddsl.Lower(doc)
```

CLI:

```bash
go run ./cmd/plano parse ./build.plano
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
- script-body execution and user-defined functions
- action validation for call statements
- example builddsl lowering and ordered task output

Run with:

```bash
go test ./...
```
