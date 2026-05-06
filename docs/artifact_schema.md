# Artifact Schema

`compiler.Artifact` is the stable serialized compiler output contract for `plano`.

Current schema version:

- `schemaVersion = "plano.artifact/v2"`

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

- `Artifact -> Result` preserves typed document, binding, check, HIR, diagnostic messages, diagnostic codes, diagnostic related-message payloads, and diagnostic suggestion title/replacement payloads.
- `Artifact -> Result` intentionally does not restore `token.FileSet` or token positions.

## v2 additions

`diagnostics[].suggestions` carries compiler-produced quick-fix metadata:

- `title`: user-facing action title
- `replacement`: replacement text for simple text edits
- `span`: source span to replace

Readers continue to accept `plano.artifact/v1` artifacts. v1 artifacts do not contain structured diagnostic suggestions.
