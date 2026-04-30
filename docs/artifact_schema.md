# Artifact Schema

`compiler.Artifact` is the stable serialized compiler output contract for `plano` `v0.3.0`.

Current schema version:

- `schemaVersion = "plano.artifact/v1"`

Top-level sections:

- `schemaVersion`
- `document`
- `binding`
- `checks`
- `hir`
- `diagnostics`

Versioning rules:

- New readers must reject unknown `schemaVersion` values.
- Readers may accept an empty `schemaVersion` only for in-memory compatibility with pre-release artifacts generated before the field existed.
- Structural or semantic breaking changes require a new schema generation, for example `plano.artifact/v2`.

Persistence guidance:

- Use `Artifact.MarshalJSON` / `Artifact.UnmarshalJSON` for portable persistence.
- Use `Artifact.MarshalBinary` / `Artifact.UnmarshalBinary` when you want the same canonical JSON payload behind a binary interface.
- Do not persist `compiler.Result` directly; it contains `token.FileSet`-backed state that is intentionally omitted from artifacts.

Round-trip expectations:

- `Artifact -> Result` preserves typed document, binding, check, HIR, diagnostic messages, diagnostic codes, and diagnostic related-message payloads.
- `Artifact -> Result` intentionally does not restore `token.FileSet` or token positions.
