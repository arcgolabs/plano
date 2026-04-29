package lsp_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcgolabs/plano/lsp"
)

func TestSnapshotCompletionAtSuggestsBuiltinsAndForms(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := fileURI(path)
	src := `
workspace {
  name = "demo"
  default = build
}

task build {
  let target = jo
}

go.
`
	if err := ws.Open(uri, 1, []byte(src)); err != nil {
		t.Fatal(err)
	}

	snapshot, err := ws.Analyze(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}

	builtinPos := positionForOffset([]byte(src), strings.Index(src, "jo")+len("jo"))
	builtinItems, ok := snapshot.CompletionAt(builtinPos)
	if !ok {
		t.Fatal("expected builtin completions")
	}
	assertCompletionContains(t, builtinItems.Items.Values(), "join_path")

	formPos := positionForOffset([]byte(src), strings.LastIndex(src, "go.")+len("go."))
	formItems, ok := snapshot.CompletionAt(formPos)
	if !ok {
		t.Fatal("expected form completions")
	}
	assertCompletionContains(t, formItems.Items.Values(), "go.binary", "go.test")
}

func TestSnapshotCompletionAtFiltersFormBodySuggestions(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := fileURI(path)
	src := `
workspace {
  na
}

task build {
  ru
  run {
    ex
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

	assertCompletionMatch(
		t,
		snapshot,
		positionForOffset([]byte(src), strings.Index(src, "na")+len("na")),
		[]string{"name"},
		[]string{"task"},
	)
	assertCompletionMatch(
		t,
		snapshot,
		positionForOffset([]byte(src), strings.Index(src, "ru\n")+len("ru")),
		[]string{"run"},
		[]string{"go.binary"},
	)
	assertCompletionMatch(
		t,
		snapshot,
		positionForOffset([]byte(src), strings.LastIndex(src, "ex")+len("ex")),
		[]string{"exec"},
		[]string{"deps"},
	)
}

func assertCompletionMatch(
	t *testing.T,
	snapshot lsp.Snapshot,
	pos lsp.Position,
	includes []string,
	excludes []string,
) {
	t.Helper()

	items, ok := snapshot.CompletionAt(pos)
	if !ok {
		t.Fatal("expected completions")
	}
	values := items.Items.Values()
	assertCompletionContains(t, values, includes...)
	assertCompletionExcludes(t, values, excludes...)
}
