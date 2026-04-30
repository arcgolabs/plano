package lsp_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/examples/builddsl"
	"github.com/arcgolabs/plano/lsp"
)

func testWorkspace(tb testing.TB) *lsp.Workspace {
	tb.Helper()
	base := compiler.New(compiler.Options{})
	if err := builddsl.Register(base); err != nil {
		tb.Fatal(err)
	}
	return lsp.NewWorkspace(lsp.Options{Compiler: base})
}

func testExprWorkspace(tb testing.TB) *lsp.Workspace {
	tb.Helper()
	base := compiler.New(compiler.Options{})
	if err := builddsl.Register(base); err != nil {
		tb.Fatal(err)
	}
	if err := base.RegisterExprVar("branch", "main"); err != nil {
		tb.Fatal(err)
	}
	if err := base.RegisterExprFunc("slug", testSlug, func(string) string { return "" }); err != nil {
		tb.Fatal(err)
	}
	return lsp.NewWorkspace(lsp.Options{Compiler: base})
}

func testSlug(params ...any) (any, error) {
	if len(params) != 1 {
		return nil, errors.New("slug expects one argument")
	}
	value, ok := params[0].(string)
	if !ok {
		return nil, errors.New("slug expects string")
	}
	return value, nil
}

func fileURI(path string) string {
	return lsp.FileURI(filepath.Clean(path))
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
	return lsp.Position{Line: line, Character: character}
}

func assertCompletionContains(t *testing.T, items []lsp.CompletionItem, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if containsCompletionLabel(items, want) {
			continue
		}
		t.Fatalf("completion items = %#v, missing %q", items, want)
	}
}

func assertCompletionExcludes(t *testing.T, items []lsp.CompletionItem, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !containsCompletionLabel(items, want) {
			continue
		}
		t.Fatalf("completion items = %#v, unexpected %q", items, want)
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

func assertDocumentSymbolNames(t *testing.T, items []lsp.DocumentSymbol, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if _, ok := findDocumentSymbol(items, want); ok {
			continue
		}
		t.Fatalf("document symbols = %#v, missing %q", items, want)
	}
}

func findDocumentSymbol(items []lsp.DocumentSymbol, want string) (lsp.DocumentSymbol, bool) {
	for index := range items {
		if items[index].Name == want {
			return items[index], true
		}
	}
	return lsp.DocumentSymbol{}, false
}
