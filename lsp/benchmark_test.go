package lsp_test

import (
	"context"
	"path/filepath"
	"strings"
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

func BenchmarkWorkspaceAnalyzeAfterUpdate(b *testing.B) {
	ws := testWorkspace(b)
	uri, src := benchmarkWorkspaceDocument(b)
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()

	variants := []string{
		strings.Replace(src, `"demo"`, `"demo-a"`, 1),
		strings.Replace(src, `"demo"`, `"demo-b"`, 1),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for index := range b.N {
		next := variants[index%len(variants)]
		if err := ws.Update(uri, int32(index+2), []byte(next)); err != nil {
			b.Fatal(err)
		}
		snapshot, err := ws.Analyze(ctx, uri)
		if err != nil {
			b.Fatal(err)
		}
		if snapshot.Diagnostics.Len() != 0 {
			b.Fatalf("unexpected diagnostics: %#v", snapshot.Diagnostics)
		}
	}
}

func BenchmarkWorkspaceAnalyzeSourceMiss(b *testing.B) {
	ws := testWorkspace(b)
	uri, src := benchmarkWorkspaceDocument(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for index := range b.N {
		snapshot, err := ws.AnalyzeSource(ctx, uri, int32(index+1), []byte(src))
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

func BenchmarkSnapshotDefinition(b *testing.B) {
	snapshot, src := benchmarkSnapshot(b)
	pos := positionOfLast(src, "target")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, ok := snapshot.DefinitionAt(pos); !ok {
			b.Fatal("expected definition result")
		}
	}
}

func BenchmarkSnapshotReferences(b *testing.B) {
	snapshot, src := benchmarkSnapshot(b)
	pos := positionOf(src, "target")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		items, ok := snapshot.ReferencesAt(pos, true)
		if !ok || items.Len() == 0 {
			b.Fatal("expected references")
		}
	}
}

func BenchmarkSnapshotDocumentSymbols(b *testing.B) {
	snapshot, _ := benchmarkSnapshot(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		items := snapshot.DocumentSymbols()
		if items.Len() == 0 {
			b.Fatal("expected document symbols")
		}
	}
}

func BenchmarkSnapshotPrepareRename(b *testing.B) {
	snapshot, src := benchmarkSnapshot(b)
	pos := positionOf(src, "target")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, ok := snapshot.PrepareRenameAt(pos); !ok {
			b.Fatal("expected prepare rename result")
		}
	}
}

func BenchmarkSnapshotCompletionTopLevel(b *testing.B) {
	snapshot, src := benchmarkSnapshotWithSource(b, `
wor
`)
	pos := positionOf(src, "wor")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		items, ok := snapshot.CompletionAt(pos)
		if !ok || items.Items.Len() == 0 {
			b.Fatal("expected top-level completion results")
		}
	}
}

func BenchmarkSnapshotExprCompletion(b *testing.B) {
	snapshot, src := benchmarkExprSnapshot(b)
	pos := positionForOffset([]byte(src), strings.Index(src, `"br"`)+len(`"br`))

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		items, ok := snapshot.CompletionAt(pos)
		if !ok || items.Items.Len() == 0 {
			b.Fatal("expected expr completion results")
		}
	}
}

func BenchmarkSnapshotExprHover(b *testing.B) {
	snapshot, src := benchmarkExprSnapshot(b)
	pos := positionOf(src, "branch")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, ok := snapshot.HoverAt(pos); !ok {
			b.Fatal("expected expr hover result")
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
	return benchmarkSnapshotWithSource(tb, benchmarkWorkspaceSource())
}

func benchmarkSnapshotWithSource(tb testing.TB, src string) (lsp.Snapshot, string) {
	tb.Helper()
	return benchmarkSnapshotWithWorkspace(tb, testWorkspace(tb), src)
}

func benchmarkSnapshotWithWorkspace(tb testing.TB, ws *lsp.Workspace, src string) (lsp.Snapshot, string) {
	tb.Helper()
	uri, _ := benchmarkWorkspaceDocument(tb)
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		tb.Fatal(err)
	}
	snapshot, err := ws.Analyze(context.Background(), uri)
	if err != nil {
		tb.Fatal(err)
	}
	return snapshot, src
}

func benchmarkExprSnapshot(tb testing.TB) (lsp.Snapshot, string) {
	tb.Helper()
	return benchmarkSnapshotWithWorkspace(tb, testExprWorkspace(tb), benchmarkExprWorkspaceSource())
}

func benchmarkWorkspaceDocument(tb testing.TB) (string, string) {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "build.plano")
	uri := fileURI(path)
	return uri, benchmarkWorkspaceSource()
}

func benchmarkWorkspaceSource() string {
	return `
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
}

func benchmarkExprWorkspaceSource() string {
	return `
workspace {
  name = "demo"
  default = build
}

task build {
  outputs = [expr("branch")]
  let next = expr("slug(branch)")
  outputs = [expr("br")]
}
`
}
