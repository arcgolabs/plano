package lsp_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/plano/lsp"
)

func TestSnapshotCodeActionsSuggestFunctionReplacement(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := fileURI(path)
	src := `
workspace {
  name = "demo"
  default = build
}

task build {
  outputs = [join_pat("dist", "demo")]
}
`
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		t.Fatal(err)
	}

	snapshot, err := ws.Analyze(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	diagnostic := firstDiagnosticWithCode(t, snapshot, "unknown-function")
	if diagnostic.Suggestions.Len() == 0 {
		t.Fatal("expected diagnostic suggestions")
	}
	actions := snapshot.CodeActions(diagnostic.Range)
	if actions.Len() == 0 {
		t.Fatal("expected code actions")
	}
	action, _ := actions.Get(0)
	if action.Title != `Replace with "join_path"` {
		t.Fatalf("action title = %q", action.Title)
	}
	edits, ok := action.Edit.Changes.Get(uri)
	if !ok {
		t.Fatalf("changes = %#v, missing %q", action.Edit.Changes, uri)
	}
	edit, _ := edits.Get(0)
	if edit.NewText != "join_path" {
		t.Fatalf("edit new text = %q", edit.NewText)
	}
	if edit.Range.Start != positionOf(src, "join_pat") {
		t.Fatalf("edit range start = %#v", edit.Range.Start)
	}
}

func TestSnapshotCodeActionsSuggestFieldReplacement(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := fileURI(path)
	src := `
workspace {
  name = "demo"
  default = build
}

task build {
  outputz = []
}
`
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		t.Fatal(err)
	}

	snapshot, err := ws.Analyze(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	diagnostic := firstDiagnosticWithCode(t, snapshot, "unknown-field")
	actions := snapshot.CodeActions(diagnostic.Range)
	if actions.Len() == 0 {
		t.Fatal("expected code actions")
	}
	action, _ := actions.Get(0)
	if action.Title != `Replace with "outputs"` {
		t.Fatalf("action title = %q", action.Title)
	}
	edits, ok := action.Edit.Changes.Get(uri)
	if !ok {
		t.Fatalf("changes = %#v, missing %q", action.Edit.Changes, uri)
	}
	edit, _ := edits.Get(0)
	if edit.NewText != "outputs" {
		t.Fatalf("edit new text = %q", edit.NewText)
	}
	if edit.Range.Start != positionOf(src, "outputz") {
		t.Fatalf("edit range start = %#v", edit.Range.Start)
	}
}

func firstDiagnosticWithCode(t *testing.T, snapshot lsp.Snapshot, code string) lsp.Diagnostic {
	t.Helper()
	diagnostics := snapshot.Diagnostics.Values()
	for index := range diagnostics {
		item := diagnostics[index]
		if item.Code == code {
			return item
		}
	}
	t.Fatalf("diagnostics = %#v, missing code %q", snapshot.Diagnostics, code)
	return lsp.Diagnostic{}
}
