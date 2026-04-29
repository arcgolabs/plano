package lsp_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/examples/builddsl"
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

func TestSnapshotCompletionAtSuggestsBuiltinsAndForms(t *testing.T) {
	ws := testWorkspace(t)
	path := filepath.Join(t.TempDir(), "build.plano")
	uri := lsp.FileURI(path)
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
	if !containsCompletionLabel(builtinItems.Items.Values(), "join_path") {
		t.Fatalf("builtin completions = %#v", builtinItems.Items.Values())
	}

	formPos := positionForOffset([]byte(src), strings.LastIndex(src, "go.")+len("go."))
	formItems, ok := snapshot.CompletionAt(formPos)
	if !ok {
		t.Fatal("expected form completions")
	}
	if !containsCompletionLabel(formItems.Items.Values(), "go.binary") {
		t.Fatalf("form completions = %#v", formItems.Items.Values())
	}
	if !containsCompletionLabel(formItems.Items.Values(), "go.test") {
		t.Fatalf("form completions = %#v", formItems.Items.Values())
	}
}

func testWorkspace(t *testing.T) *lsp.Workspace {
	t.Helper()
	base := compiler.New(compiler.Options{})
	if err := builddsl.Register(base); err != nil {
		t.Fatal(err)
	}
	return lsp.NewWorkspace(lsp.Options{Compiler: base})
}

func positionOf(src, needle string) lsp.Position {
	index := strings.Index(src, needle)
	if index < 0 {
		panic("missing needle: " + needle)
	}
	return positionForOffset([]byte(src), index)
}

func positionOfLast(src, needle string) lsp.Position {
	index := strings.LastIndex(src, needle)
	if index < 0 {
		panic("missing needle: " + needle)
	}
	return positionForOffset([]byte(src), index)
}

func positionForOffset(src []byte, offset int) lsp.Position {
	line := 0
	character := 0
	for idx := 0; idx < offset; {
		if src[idx] == '\n' {
			line++
			character = 0
			idx++
			continue
		}
		r, size := utf8.DecodeRune(src[idx:])
		if r == utf8.RuneError {
			character++
		} else {
			character += len(utf16.Encode([]rune{r}))
		}
		idx += size
	}
	return lsp.Position{
		Line:      line,
		Character: character,
	}
}

func containsCompletionLabel(items []lsp.CompletionItem, want string) bool {
	for _, item := range items {
		if item.Label == want {
			return true
		}
	}
	return false
}
