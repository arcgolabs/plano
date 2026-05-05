package lsp_test

import (
	"testing"

	"github.com/arcgolabs/plano/lsp"
)

func BenchmarkSnapshotCodeAction(b *testing.B) {
	snapshot, src := benchmarkSnapshotWithSource(b, `
workspace {
  name = "demo"
  default = build
}

task build {
  outputs = [join_pat("dist", "demo")]
}
`)
	pos := positionOf(src, "join_pat")
	rng := lsp.Range{Start: pos, End: pos}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		actions := snapshot.CodeActions(rng)
		if actions.Len() == 0 {
			b.Fatal("expected code actions")
		}
	}
}
