package lsp_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/lsp"
)

func TestSnapshotFoldingRangesIncludeFormsAndBlocks(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := fileURI(path)
	src := `
fn output(name: string): path {
  return join_path("dist", name)
}

workspace {
  name = "demo"
  default = build
}

task build {
  if true {
    outputs = [output("demo")]
  }
}
`
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		t.Fatal(err)
	}

	snapshot, err := ws.Analyze(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	ranges := snapshot.FoldingRanges()
	if ranges.Len() < 4 {
		t.Fatalf("folding ranges = %#v", ranges)
	}
	assertFoldingRangeStartsAt(t, ranges, positionOf(src, "fn output"))
	assertFoldingRangeStartsAt(t, ranges, positionOf(src, "workspace"))
	assertFoldingRangeStartsAt(t, ranges, positionOf(src, "task build"))
	assertFoldingRangeStartsOnLine(t, ranges, positionOf(src, "if true").Line)
}

func assertFoldingRangeStartsAt(t *testing.T, ranges list.List[lsp.FoldingRange], want lsp.Position) {
	t.Helper()
	for _, item := range ranges.Values() {
		if item.Range.Start == want {
			return
		}
	}
	t.Fatalf("folding ranges = %#v, missing start %#v", ranges, want)
}

func assertFoldingRangeStartsOnLine(t *testing.T, ranges list.List[lsp.FoldingRange], line int) {
	t.Helper()
	for _, item := range ranges.Values() {
		if item.Range.Start.Line == line {
			return
		}
	}
	t.Fatalf("folding ranges = %#v, missing start line %d", ranges, line)
}
