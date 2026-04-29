package lsp_test

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSnapshotReferencesAtFindsSymbolReferences(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := fileURI(path)
	src := `
workspace {
  name = "demo"
  default = build
}

task build {}
`
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		t.Fatal(err)
	}

	snapshot, err := ws.Analyze(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	refs, ok := snapshot.ReferencesAt(positionOf(src, "build {}"), true)
	if !ok {
		t.Fatal("expected references")
	}
	if refs.Len() != 2 {
		t.Fatalf("references = %#v", refs.Values())
	}
}

func TestSnapshotDocumentSymbolsIncludesFormsAndFields(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := fileURI(path)
	src := `
const project_name: string = "demo"

fn output_dir(name: string): path {
  return join_path("dist", name)
}

task build {
  outputs = [output_dir(project_name)]
  run {
    exec("go", "test", "./...")
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
	symbols := snapshot.DocumentSymbols()
	if symbols.Len() < 3 {
		t.Fatalf("document symbols = %#v", symbols.Values())
	}
	assertDocumentSymbolNames(t, symbols.Values(), "project_name", "output_dir", "build")

	buildSymbol, ok := findDocumentSymbol(symbols.Values(), "build")
	if !ok {
		t.Fatalf("document symbols = %#v", symbols.Values())
	}
	assertDocumentSymbolNames(t, buildSymbol.Children.Values(), "outputs", "run")
}
