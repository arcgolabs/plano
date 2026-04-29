package lsp_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcgolabs/plano/lsp"
)

func TestWorkspaceAnalyzeUsesOpenDocumentsForImports(t *testing.T) {
	rootDir := t.TempDir()
	defsPath := filepath.Join(rootDir, "defs.plano")
	rootPath := filepath.Join(rootDir, "build.plano")

	if err := os.WriteFile(defsPath, []byte(`const project_name: string = 1`), 0o600); err != nil {
		t.Fatal(err)
	}
	rootSource := `
import "./defs.plano"

workspace {
  name = project_name
  default = build
}

task build {}
`
	if err := os.WriteFile(rootPath, []byte(rootSource), 0o600); err != nil {
		t.Fatal(err)
	}

	ws := testWorkspace(t)
	if err := ws.Open(lsp.FileURI(defsPath), 2, []byte(`const project_name: string = "demo"`)); err != nil {
		t.Fatal(err)
	}

	snapshot, err := ws.Analyze(context.Background(), lsp.FileURI(rootPath))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Diagnostics.Len() != 0 {
		t.Fatalf("diagnostics = %#v", snapshot.Diagnostics)
	}
}

func TestSnapshotDefinitionAtFindsLocalBinding(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := lsp.FileURI(path)
	src := `
workspace {
  name = "demo"
  default = build
}

task build {
  let target = join_path("dist", "demo")
  outputs = [target]
}
`
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		t.Fatal(err)
	}

	snapshot, err := ws.Analyze(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	location, ok := snapshot.DefinitionAt(positionOfLast(src, "target"))
	if !ok {
		t.Fatal("expected definition")
	}
	if location.URI != uri {
		t.Fatalf("definition uri = %q, want %q", location.URI, uri)
	}
	expected := positionOf(src, "target = join_path")
	if location.Range.Start != expected {
		t.Fatalf("definition start = %#v, want %#v", location.Range.Start, expected)
	}
}

func TestSnapshotHoverAtShowsBuiltinDocsAndTypes(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := lsp.FileURI(path)
	src := `
workspace {
  name = "demo"
  default = build
}

task build {
  let target = join_path("dist", "demo")
  outputs = [target]
}
`
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		t.Fatal(err)
	}

	snapshot, err := ws.Analyze(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	hover, ok := snapshot.HoverAt(positionOf(src, "join_path"))
	if !ok {
		t.Fatal("expected hover")
	}
	if !strings.Contains(hover.Contents, "fn join_path") {
		t.Fatalf("hover = %q", hover.Contents)
	}
	if !strings.Contains(hover.Contents, "normalized path") {
		t.Fatalf("hover = %q", hover.Contents)
	}
	if !strings.Contains(hover.Contents, "type: `path`") {
		t.Fatalf("hover = %q", hover.Contents)
	}
}
