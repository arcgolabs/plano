//nolint:testpackage,cyclop,gocognit,gocyclo,revive // Compiler tests stay in-package to exercise helpers and internal state.
package compiler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/plano/schema"
)

func TestCompileSource(t *testing.T) {
	c := newTestCompiler()
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
	if got := len(doc.Forms); got != 3 {
		t.Fatalf("forms = %d, want 3", got)
	}
	if got := doc.Consts["target"]; got != filepath.Join("dist", "app") {
		t.Fatalf("target const = %#v, want dist/app", got)
	}

	workspace := doc.Forms[0]
	if workspace.Kind != "workspace" {
		t.Fatalf("workspace kind = %q", workspace.Kind)
	}
	if got := workspace.Fields["default"]; got != (schema.Ref{Kind: "task", Name: "build"}) {
		t.Fatalf("workspace default = %#v", got)
	}

	build := doc.Forms[1]
	deps, ok := build.Fields["deps"].([]any)
	if !ok || len(deps) != 1 {
		t.Fatalf("deps = %#v, want one ref", build.Fields["deps"])
	}
	if got := deps[0]; got != (schema.Ref{Kind: "task", Name: "prepare"}) {
		t.Fatalf("deps[0] = %#v", got)
	}
	if got := len(build.Forms); got != 1 {
		t.Fatalf("nested forms = %d, want 1", got)
	}
	if got := len(build.Forms[0].Calls); got != 1 {
		t.Fatalf("run calls = %d, want 1", got)
	}
	if got := build.Forms[0].Calls[0].Name; got != "exec" {
		t.Fatalf("call name = %q, want exec", got)
	}
}

func TestCompileFileWithImport(t *testing.T) {
	c := newTestCompiler()
	dir := t.TempDir()

	root := filepath.Join(dir, "build.plano")
	taskFile := filepath.Join(dir, "tasks.plano")

	if err := os.WriteFile(taskFile, []byte(`
task prepare {}
`), 0o600); err != nil {
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
	if got := len(doc.Forms); got != 2 {
		t.Fatalf("forms = %d, want 2", got)
	}
	if got := doc.Forms[0].Kind; got != "task" {
		t.Fatalf("first imported form kind = %q, want task", got)
	}
}

func TestCompileFileWithGlobImport(t *testing.T) {
	c := newTestCompiler()
	dir := t.TempDir()

	root := filepath.Join(dir, "build.plano")
	if err := os.MkdirAll(filepath.Join(dir, "tasks", "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks", "prepare.plano"), []byte(`
task prepare {}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks", "nested", "test.plano"), []byte(`
task test {}
`), 0o600); err != nil {
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
	if got := len(doc.Forms); got != 3 {
		t.Fatalf("forms = %d, want 3", got)
	}
}

func TestCompileUnknownReference(t *testing.T) {
	c := newTestCompiler()
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
	c := newTestCompiler()
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
	c := newTestCompiler()
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
	if got := len(doc.Forms); got != 1 {
		t.Fatalf("forms = %d, want 1", got)
	}
	build := doc.Forms[0]
	outputs, ok := build.Fields["outputs"].([]any)
	if !ok || len(outputs) != 1 {
		t.Fatalf("outputs = %#v, want one path", build.Fields["outputs"])
	}
	if got := outputs[0]; got != filepath.Join("dist", "app") {
		t.Fatalf("output path = %#v", got)
	}
	if got := len(build.Forms); got != 2 {
		t.Fatalf("nested run forms = %d, want 2", got)
	}
	if got := build.Forms[0].Calls[0].Args[2]; got != "./..." {
		t.Fatalf("first run arg = %#v, want ./...", got)
	}
	if got := build.Forms[1].Calls[0].Args[2]; got != "./cmd/..." {
		t.Fatalf("second run arg = %#v, want ./cmd/...", got)
	}
}

func newTestCompiler() *Compiler {
	c := New(Options{
		LookupEnv: func(string) (string, bool) { return "", false },
	})
	mustRegister(c, schema.FormSpec{
		Name:      "workspace",
		LabelKind: schema.LabelNone,
		BodyMode:  schema.BodyFieldOnly,
		Fields: map[string]schema.FieldSpec{
			"name": {
				Name:     "name",
				Type:     schema.TypeString,
				Required: true,
			},
			"default": {
				Name:     "default",
				Type:     schema.RefType{Kind: "task"},
				Required: true,
			},
		},
	})
	mustRegister(c, schema.FormSpec{
		Name:         "task",
		LabelKind:    schema.LabelSymbol,
		BodyMode:     schema.BodyScript,
		LabelRefKind: "task",
		Declares:     "task",
		Fields: map[string]schema.FieldSpec{
			"deps": {
				Name:       "deps",
				Type:       schema.ListType{Elem: schema.RefType{Kind: "task"}},
				Default:    []any{},
				HasDefault: true,
			},
			"outputs": {
				Name:       "outputs",
				Type:       schema.ListType{Elem: schema.TypePath},
				Default:    []any{},
				HasDefault: true,
			},
		},
		NestedForms: map[string]struct{}{
			"run": {},
		},
	})
	mustRegister(c, schema.FormSpec{
		Name:      "run",
		LabelKind: schema.LabelNone,
		BodyMode:  schema.BodyCallOnly,
	})
	mustRegisterAction(c, ActionSpec{
		Name:    "exec",
		MinArgs: 1,
		MaxArgs: -1,
		Validate: func(args []any) error {
			for _, arg := range args {
				if _, ok := arg.(string); !ok {
					return errors.New("exec expects string arguments")
				}
			}
			return nil
		},
	})
	return c
}

func mustRegister(c *Compiler, spec schema.FormSpec) {
	if err := c.RegisterForm(spec); err != nil {
		panic(err)
	}
}

func mustRegisterAction(c *Compiler, spec ActionSpec) {
	if err := c.RegisterAction(spec); err != nil {
		panic(err)
	}
}
