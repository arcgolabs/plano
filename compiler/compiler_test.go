package compiler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
)

func TestCompileSource(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
const output: string = target
const target: string = join_path("dist", "app")

workspace {
  name = "demo"
  default = build
}

task build {
  deps = [prepare]
  outputs = [output]

  run {
    exec("go", "build", "./...")
  }
}

task prepare {}
`)

	doc, diags := c.CompileSource(context.Background(), "build.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	assertFormCount(t, doc, 3)
	assertTargetConst(t, doc)
	assertWorkspaceDefault(t, formAt(t, doc.Forms, 0), "build")
	assertTaskDependencies(t, formAt(t, doc.Forms, 1), []schema.Ref{{Kind: "task", Name: "prepare"}})
	assertFirstNestedCall(t, formAt(t, doc.Forms, 1), "exec")
}

func TestCompileFileWithImport(t *testing.T) {
	c := newTestCompiler(t)
	dir := t.TempDir()

	root := filepath.Join(dir, "build.plano")
	taskFile := filepath.Join(dir, "tasks.plano")

	if err := os.WriteFile(taskFile, []byte(`task prepare {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(`
import "tasks.plano"

workspace {
  name = "demo"
  default = prepare
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	doc, diags := c.CompileFile(context.Background(), root)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := doc.Forms.Len(); got != 2 {
		t.Fatalf("forms = %d, want 2", got)
	}
	if got := formAt(t, doc.Forms, 0).Kind; got != "task" {
		t.Fatalf("first imported form kind = %q, want task", got)
	}
}

func TestCompileFileWithGlobImport(t *testing.T) {
	c := newTestCompiler(t)
	dir := t.TempDir()

	root := filepath.Join(dir, "build.plano")
	if err := os.MkdirAll(filepath.Join(dir, "tasks", "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks", "prepare.plano"), []byte(`task prepare {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks", "nested", "test.plano"), []byte(`task test {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte(`
import "tasks/**/*.plano"

workspace {
  name = "demo"
  default = test
}
`), 0o600); err != nil {
		t.Fatal(err)
	}

	doc, diags := c.CompileFile(context.Background(), root)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := doc.Forms.Len(); got != 3 {
		t.Fatalf("forms = %d, want 3", got)
	}
}

func TestCompileUnknownReference(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  deps = [missing]
}
`)

	_, diags := c.CompileSource(context.Background(), "build.plano", src)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for unknown symbol")
	}
}

func TestCompileUnknownAction(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  run {
    missing("x")
  }
}
`)

	_, diags := c.CompileSource(context.Background(), "build.plano", src)
	if !diags.HasError() {
		t.Fatal("expected diagnostics for unknown action")
	}
}

func TestCompileScriptBodyAndFunctions(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
fn output(name: string): path {
  return join_path("dist", name)
}

task build {
  let packages = ["./...", "./cmd/..."]
  let include = true

  if include {
    outputs = [output("app")]
  }

  for pkg in packages {
    run {
      exec("go", "test", pkg)
    }
  }
}
`)

	doc, diags := c.CompileSource(context.Background(), "build.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	assertFormCount(t, doc, 1)
	build := formAt(t, doc.Forms, 0)
	assertTaskOutputs(t, build, []string{filepath.Join("dist", "app")})
	assertCallArgs(t, firstCall(t, nestedFormAt(t, build, 0)).Args, "./...")
	assertCallArgs(t, firstCall(t, nestedFormAt(t, build, 1)).Args, "./cmd/...")
}

func assertFormCount(t *testing.T, doc *compiler.Document, want int) {
	t.Helper()
	if got := doc.Forms.Len(); got != want {
		t.Fatalf("forms = %d, want %d", got, want)
	}
}

func assertTargetConst(t *testing.T, doc *compiler.Document) {
	t.Helper()
	target, ok := doc.Const("target")
	if !ok || target != filepath.Join("dist", "app") {
		t.Fatalf("target const = %#v, want dist/app", target)
	}
}

func assertWorkspaceDefault(t *testing.T, form compiler.Form, want string) {
	t.Helper()
	if form.Kind != "workspace" {
		t.Fatalf("workspace kind = %q", form.Kind)
	}
	defaultRef, ok := form.Field("default")
	if !ok || defaultRef != (schema.Ref{Kind: "task", Name: want}) {
		t.Fatalf("workspace default = %#v", defaultRef)
	}
}

func assertTaskDependencies(t *testing.T, form compiler.Form, want []schema.Ref) {
	t.Helper()
	depsValue, ok := form.Field("deps")
	if !ok {
		t.Fatal("deps missing")
	}
	deps, ok := depsValue.([]any)
	if !ok || len(deps) != len(want) {
		t.Fatalf("deps = %#v, want %#v", depsValue, want)
	}
	for idx, item := range want {
		if deps[idx] != item {
			t.Fatalf("deps[%d] = %#v, want %#v", idx, deps[idx], item)
		}
	}
}

func assertFirstNestedCall(t *testing.T, form compiler.Form, want string) {
	t.Helper()
	if form.Forms.Len() != 1 {
		t.Fatalf("nested forms = %d, want 1", form.Forms.Len())
	}
	run := nestedFormAt(t, form, 0)
	if run.Calls.Len() != 1 {
		t.Fatalf("run calls = %d, want 1", run.Calls.Len())
	}
	if got := firstCall(t, run).Name; got != want {
		t.Fatalf("call name = %q, want %q", got, want)
	}
}

func assertTaskOutputs(t *testing.T, form compiler.Form, want []string) {
	t.Helper()
	outputsValue, ok := form.Field("outputs")
	if !ok {
		t.Fatal("outputs missing")
	}
	outputs, ok := outputsValue.([]any)
	if !ok || len(outputs) != len(want) {
		t.Fatalf("outputs = %#v, want %#v", outputsValue, want)
	}
	for idx, item := range want {
		if outputs[idx] != item {
			t.Fatalf("outputs[%d] = %#v, want %#v", idx, outputs[idx], item)
		}
	}
}

func assertCallArgs(t *testing.T, args list.List[any], wantLast string) {
	t.Helper()
	values := args.Values()
	if len(values) < 3 {
		t.Fatalf("args = %#v", values)
	}
	if got := values[2]; got != wantLast {
		t.Fatalf("call arg = %#v, want %#v", got, wantLast)
	}
}

func formAt(t *testing.T, forms list.List[compiler.Form], index int) compiler.Form {
	t.Helper()
	form, ok := forms.Get(index)
	if !ok {
		t.Fatalf("form %d missing", index)
	}
	return form
}

func nestedFormAt(t *testing.T, form compiler.Form, index int) compiler.Form {
	t.Helper()
	nested, ok := form.Forms.Get(index)
	if !ok {
		t.Fatalf("nested form %d missing", index)
	}
	return nested
}

func firstCall(t *testing.T, form compiler.Form) compiler.Call {
	t.Helper()
	call, ok := form.Calls.Get(0)
	if !ok {
		t.Fatal("first call missing")
	}
	return call
}
