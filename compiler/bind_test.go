package compiler_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
)

func TestBindSourceDetailed(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
const target: string = "dist/demo"

fn output(name: string): path {
  return name
}

task prepare {}
task build {}
`)

	result := c.BindSourceDetailed(context.Background(), "build.plano", src)
	if result.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if result.Binding == nil {
		t.Fatal("expected binding")
	}
	assertStrings(t, "files", result.Binding.Files, []string{"build.plano"})
	assertStrings(t, "const order", result.Binding.Consts.Keys(), []string{"target"})
	assertConstType(t, result.Binding, "target", schema.TypeString.String())
	assertStrings(t, "function order", result.Binding.Functions.Keys(), []string{"output"})
	assertFunctionSignature(t, result.Binding, "output", []string{schema.TypeString.String()}, schema.TypePath.String())
	assertStrings(t, "symbol order", result.Binding.Symbols.Keys(), []string{"prepare", "build"})
	if got := result.Binding.Locals.Len(); got != 1 {
		t.Fatalf("local count = %d", got)
	}
	assertLocalNames(t, result.Binding, []string{"name"})
	assertUseKinds(t, result.Binding, map[string]compiler.NameUseKind{
		"name": compiler.UseLocal,
	})
}

func TestBindFileDetailedWithImportOrder(t *testing.T) {
	c := newTestCompiler(t)
	dir := t.TempDir()

	root := filepath.Join(dir, "build.plano")
	taskFile := filepath.Join(dir, "tasks.plano")

	if err := os.WriteFile(taskFile, []byte(`task prepare {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(`
import "tasks.plano"

task build {}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	result := c.BindFileDetailed(context.Background(), root)
	if result.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if got := result.Binding.Symbols.Keys(); !reflect.DeepEqual(got, []string{"prepare", "build"}) {
		t.Fatalf("symbol order = %#v", got)
	}
	if got := len(result.Binding.Files); got != 2 {
		t.Fatalf("file count = %d", got)
	}
}

func TestBindDuplicateDefinition(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
const build: string = "x"
task build {}
`)

	result := c.BindSourceDetailed(context.Background(), "build.plano", src)
	if !result.Diagnostics.HasError() {
		t.Fatal("expected duplicate definition diagnostics")
	}
}

func TestBindResolvesScopesAndUses(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
const target: string = "dist/demo"

fn output(name: string): path {
  let suffix = name
  if true {
    let nested = suffix
    return nested
  }
  return name
}

task build {
  let packages = ["./..."]
  for pkg in packages {
    run {
      exec("go", "test", pkg)
    }
  }
  outputs = [target, output("demo"), os]
}
`)

	result := c.BindSourceDetailed(context.Background(), "build.plano", src)
	if result.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if got := result.Binding.Scopes.Len(); got < 6 {
		t.Fatalf("scope count = %d", got)
	}
	assertLocalNames(t, result.Binding, []string{"name", "suffix", "nested", "packages", "pkg"})
	assertUseKinds(t, result.Binding, map[string]compiler.NameUseKind{
		"name":     compiler.UseLocal,
		"suffix":   compiler.UseLocal,
		"nested":   compiler.UseLocal,
		"packages": compiler.UseLocal,
		"pkg":      compiler.UseLocal,
		"target":   compiler.UseConst,
		"output":   compiler.UseFunction,
		"os":       compiler.UseGlobal,
		"exec":     compiler.UseAction,
	})
}

func assertConstType(t *testing.T, binding *compiler.Binding, name, want string) {
	t.Helper()
	item, ok := binding.Consts.Get(name)
	if !ok {
		t.Fatalf("expected const binding %q", name)
	}
	if got := item.Type.String(); got != want {
		t.Fatalf("const %q type = %q", name, got)
	}
}

func assertFunctionSignature(t *testing.T, binding *compiler.Binding, name string, wantParams []string, wantResult string) {
	t.Helper()
	item, ok := binding.Functions.Get(name)
	if !ok {
		t.Fatalf("expected function binding %q", name)
	}
	gotParams := make([]string, 0, len(item.Params))
	for _, param := range item.Params {
		gotParams = append(gotParams, param.Type.String())
	}
	assertStrings(t, "function params", gotParams, wantParams)
	if got := item.Result.String(); got != wantResult {
		t.Fatalf("function %q result = %q", name, got)
	}
}

func assertStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v", label, got)
	}
}

func assertLocalNames(t *testing.T, binding *compiler.Binding, want []string) {
	t.Helper()
	names := make([]string, 0, binding.Locals.Len())
	for _, item := range binding.Locals.Values() {
		names = append(names, item.Name)
	}
	assertStrings(t, "local names", names, want)
}

func assertUseKinds(t *testing.T, binding *compiler.Binding, want map[string]compiler.NameUseKind) {
	t.Helper()
	got := make(map[string]compiler.NameUseKind)
	for _, item := range binding.Uses.Values() {
		if _, ok := got[item.Name]; !ok {
			got[item.Name] = item.Kind
		}
	}
	for name, kind := range want {
		if got[name] != kind {
			t.Fatalf("use %q kind = %q, want %q", name, got[name], kind)
		}
	}
}
