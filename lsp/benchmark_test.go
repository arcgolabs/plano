package lsp_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/plano/lsp"
)

func BenchmarkWorkspaceAnalyze(b *testing.B) {
	ws := testWorkspace(b)
	uri, src := benchmarkWorkspaceDocument(b)
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		snapshot, err := ws.Analyze(ctx, uri)
		if err != nil {
			b.Fatal(err)
		}
		if snapshot.Diagnostics.Len() != 0 {
			b.Fatalf("unexpected diagnostics: %#v", snapshot.Diagnostics)
		}
	}
}

func BenchmarkSnapshotHover(b *testing.B) {
	snapshot, src := benchmarkSnapshot(b)
	pos := positionOf(src, "join_path")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, ok := snapshot.HoverAt(pos); !ok {
			b.Fatal("expected hover result")
		}
	}
}

func BenchmarkSnapshotCompletion(b *testing.B) {
	snapshot, src := benchmarkSnapshot(b)
	pos := positionOf(src, "join_path")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		items, ok := snapshot.CompletionAt(pos)
		if !ok || items.Items.Len() == 0 {
			b.Fatal("expected completion results")
		}
	}
}

func BenchmarkSnapshotRename(b *testing.B) {
	snapshot, src := benchmarkSnapshot(b)
	pos := positionOf(src, "target")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		edit, ok := snapshot.RenameAt(pos, "artifact_path")
		if !ok || edit.Changes == nil || edit.Changes.Len() == 0 {
			b.Fatal("expected rename edits")
		}
	}
}

func benchmarkSnapshot(tb testing.TB) (lsp.Snapshot, string) {
	tb.Helper()
	ws := testWorkspace(tb)
	uri, src := benchmarkWorkspaceDocument(tb)
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		tb.Fatal(err)
	}
	snapshot, err := ws.Analyze(context.Background(), uri)
	if err != nil {
		tb.Fatal(err)
	}
	return snapshot, src
}

func benchmarkWorkspaceDocument(tb testing.TB) (string, string) {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "build.plano")
	uri := fileURI(path)
	src := `
workspace {
  name = "demo"
  default = build
}

task build {
  let target = join_path("dist", "demo")
  outputs = [target]

  run {
    exec("go", "test", "./...")
  }
}
`
	return uri, src
}
