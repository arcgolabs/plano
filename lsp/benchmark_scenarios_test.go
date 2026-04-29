package lsp_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcgolabs/plano/lsp"
)

func BenchmarkWorkspaceAnalyzeLargeAfterUpdate(b *testing.B) {
	ws := testWorkspace(b)
	src := benchmarkWorkspaceLargeSource(20)
	path := filepath.Join(b.TempDir(), "build.plano")
	uri := fileURI(path)
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

func BenchmarkSnapshotReferencesDenseSymbol(b *testing.B) {
	snapshot, src := benchmarkDenseReferenceSnapshot(b)
	pos := positionOf(src, "artifact_name")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		items, ok := snapshot.ReferencesAt(pos, true)
		if !ok || items.Len() == 0 {
			b.Fatal("expected dense references")
		}
	}
}

func BenchmarkSnapshotDocumentSymbolsLargeDocument(b *testing.B) {
	snapshot, _ := benchmarkLargeSnapshot(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		items := snapshot.DocumentSymbols()
		if items.Len() == 0 {
			b.Fatal("expected document symbols")
		}
	}
}

func BenchmarkSnapshotCompletionFieldContext(b *testing.B) {
	src := benchmarkCompletionFieldSource(12)
	snapshot, analyzed := benchmarkSnapshotWithSource(b, src)
	index := strings.Index(analyzed, "\n  ou\n")
	if index < 0 {
		b.Fatal("missing field completion marker")
	}
	pos := positionForOffset([]byte(analyzed), index+len("\n  ou"))

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		items, ok := snapshot.CompletionAt(pos)
		if !ok || items.Items.Len() == 0 {
			b.Fatal("expected field completions")
		}
	}
}

func benchmarkLargeSnapshot(tb testing.TB) (lsp.Snapshot, string) {
	tb.Helper()
	return benchmarkSnapshotWithSource(tb, benchmarkWorkspaceLargeSource(20))
}

func benchmarkDenseReferenceSnapshot(tb testing.TB) (lsp.Snapshot, string) {
	tb.Helper()
	return benchmarkSnapshotWithSource(tb, benchmarkDenseReferenceSource(32))
}

func benchmarkWorkspaceLargeSource(taskCount int) string {
	var builder strings.Builder
	mustWriteString(&builder, `
const artifact_name: string = "demo"

fn output(name: string): path {
  return join_path("dist", name)
}

fn packages(): list<string> {
  let base = append(["./..."], "./cmd/...")
  if has(merge({unit = "./..."}, {cli = "./cmd/..."}), "cli") {
    base = concat(base, ["./internal/..."])
  }
  return base
}

workspace {
  name = artifact_name
  default = build_00
}
`)
	for index := range taskCount {
		deps := "[]"
		if index > 0 {
			deps = fmt.Sprintf("[build_%02d]", index-1)
		}
		mustFprintf(&builder, `
task build_%02d {
  deps = %s
  let outputs_map = merge(
    {primary = output("artifact_%02d")},
    {backup = output("artifact_%02d_backup")},
  )
  outputs = values(outputs_map)

  for idx, pkg in packages() {
    if idx == 1 {
      continue
    }
    if has(["./internal/..."], pkg) {
      break
    }
    run {
      exec("go", "test", pkg)
    }
  }

  for label, artifact in outputs_map {
    if label == "backup" {
      break
    }
    run {
      exec("echo", artifact)
    }
  }
}
`, index, deps, index, index)
	}
	return builder.String()
}

func benchmarkDenseReferenceSource(taskCount int) string {
	var builder strings.Builder
	mustWriteString(&builder, `
const artifact_name: string = "demo"

workspace {
  name = artifact_name
  default = build_00
}
`)
	for index := range taskCount {
		deps := "[]"
		if index > 0 {
			deps = fmt.Sprintf("[build_%02d]", index-1)
		}
		mustFprintf(&builder, `
task build_%02d {
  deps = %s
  outputs = [
    join_path("dist", artifact_name),
    join_path("dist", artifact_name),
  ]
}
`, index, deps)
	}
	return builder.String()
}

func benchmarkCompletionFieldSource(taskCount int) string {
	src := benchmarkWorkspaceLargeSource(taskCount)
	return strings.Replace(src, "outputs = values(outputs_map)", "ou", 1)
}

func mustWriteString(builder *strings.Builder, value string) {
	if _, err := builder.WriteString(value); err != nil {
		panic(err)
	}
}

func mustFprintf(builder *strings.Builder, format string, args ...any) {
	if _, err := fmt.Fprintf(builder, format, args...); err != nil {
		panic(err)
	}
}
