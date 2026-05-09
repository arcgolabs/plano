package lsp_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/lsp"
	"github.com/arcgolabs/plano/schema"
)

func testWorkspace(tb testing.TB) *lsp.Workspace {
	tb.Helper()
	base := compiler.New(compiler.Options{})
	if err := registerTestBuildDSL(base); err != nil {
		tb.Fatal(err)
	}
	return lsp.NewWorkspace(lsp.Options{Compiler: base})
}

func testExprWorkspace(tb testing.TB) *lsp.Workspace {
	tb.Helper()
	base := compiler.New(compiler.Options{})
	if err := registerTestBuildDSL(base); err != nil {
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

func registerTestBuildDSL(c *compiler.Compiler) error {
	if err := c.RegisterForms(schema.FormSpecs(
		schema.FormSpec{
			Name:      "workspace",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyFieldOnly,
			Fields: schema.Fields(
				schema.FieldSpec{Name: "name", Type: schema.TypeString, Required: true},
				schema.FieldSpec{Name: "default", Type: schema.RefType{Kind: "task"}, Required: true},
			),
		},
		schema.FormSpec{
			Name:         "task",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "task",
			BodyMode:     schema.BodyScript,
			Declares:     "task",
			Fields: schema.Fields(
				schema.FieldSpec{
					Name:       "deps",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "task"}},
					Default:    []any{},
					HasDefault: true,
				},
				schema.FieldSpec{
					Name:       "outputs",
					Type:       schema.ListType{Elem: schema.TypePath},
					Default:    []any{},
					HasDefault: true,
				},
			),
			NestedForms: schema.NestedForms("run"),
		},
		schema.FormSpec{
			Name:         "go.test",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "task",
			BodyMode:     schema.BodyFieldOnly,
			Declares:     "task",
			Fields: schema.Fields(
				schema.FieldSpec{
					Name:       "deps",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "task"}},
					Default:    []any{},
					HasDefault: true,
				},
				schema.FieldSpec{Name: "packages", Type: schema.ListType{Elem: schema.TypePath}, Required: true},
			),
		},
		schema.FormSpec{
			Name:         "go.binary",
			LabelKind:    schema.LabelSymbol,
			LabelRefKind: "task",
			BodyMode:     schema.BodyFieldOnly,
			Declares:     "task",
			Fields: schema.Fields(
				schema.FieldSpec{
					Name:       "deps",
					Type:       schema.ListType{Elem: schema.RefType{Kind: "task"}},
					Default:    []any{},
					HasDefault: true,
				},
				schema.FieldSpec{Name: "main", Type: schema.TypePath, Required: true},
				schema.FieldSpec{Name: "out", Type: schema.TypePath, Required: true},
			),
		},
		schema.FormSpec{
			Name:      "run",
			LabelKind: schema.LabelNone,
			BodyMode:  schema.BodyCallOnly,
		},
	)); err != nil {
		return err
	}
	return c.RegisterActions(compiler.ActionSpecs(
		compiler.ActionSpec{
			Name:         "exec",
			MinArgs:      1,
			MaxArgs:      -1,
			ArgTypes:     schema.Types(schema.TypeString),
			VariadicType: schema.TypeString,
			Validate:     validateTestStringArgs,
		},
		compiler.ActionSpec{
			Name:     "shell",
			MinArgs:  1,
			MaxArgs:  1,
			ArgTypes: schema.Types(schema.TypeString),
			Validate: validateTestStringArgs,
		},
	))
}

func validateTestStringArgs(args list.List[any]) error {
	for _, arg := range args.Values() {
		if _, ok := arg.(string); !ok {
			return errors.New("test build action expects string arguments")
		}
	}
	return nil
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
