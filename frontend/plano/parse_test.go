//nolint:testpackage,cyclop,gocognit,gocyclo // Parser tests stay in-package for concise AST assertions.
package plano

import (
	"go/token"
	"testing"

	"github.com/arcgolabs/plano/ast"
)

func TestParseFile(t *testing.T) {
	src := []byte(`
const target: string = os + "/" + arch

workspace {
  name = "demo"
  default = build
}

task build {
  deps = [prepare]

  run {
    exec("go", "build", "./...")
  }
}
`)

	file, diags := ParseFile(token.NewFileSet(), "build.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got := len(file.Statements); got != 3 {
		t.Fatalf("expected 3 statements, got %d", got)
	}

	constDecl, ok := file.Statements[0].(*ast.ConstDecl)
	if !ok {
		t.Fatalf("statement 0 = %T, want *ast.ConstDecl", file.Statements[0])
	}
	if constDecl.Name.Name != "target" {
		t.Fatalf("const name = %q, want target", constDecl.Name.Name)
	}

	workspace, ok := file.Statements[1].(*ast.FormDecl)
	if !ok {
		t.Fatalf("statement 1 = %T, want *ast.FormDecl", file.Statements[1])
	}
	if got := workspace.Head.String(); got != "workspace" {
		t.Fatalf("workspace head = %q, want workspace", got)
	}
	if workspace.Label != nil {
		t.Fatalf("workspace label = %#v, want nil", workspace.Label)
	}
	if got := len(workspace.Body.Items); got != 2 {
		t.Fatalf("workspace body items = %d, want 2", got)
	}

	task, ok := file.Statements[2].(*ast.FormDecl)
	if !ok {
		t.Fatalf("statement 2 = %T, want *ast.FormDecl", file.Statements[2])
	}
	if got := task.Head.String(); got != "task" {
		t.Fatalf("task head = %q, want task", got)
	}
	if task.Label == nil || task.Label.Value != "build" {
		t.Fatalf("task label = %#v, want build", task.Label)
	}
	if got := len(task.Body.Items); got != 2 {
		t.Fatalf("task body items = %d, want 2", got)
	}
	runForm, ok := task.Body.Items[1].(*ast.FormDecl)
	if !ok {
		t.Fatalf("task body item 1 = %T, want *ast.FormDecl", task.Body.Items[1])
	}
	if got := runForm.Head.String(); got != "run" {
		t.Fatalf("run form head = %q, want run", got)
	}
	call, ok := runForm.Body.Items[0].(*ast.CallStmt)
	if !ok {
		t.Fatalf("run body item 0 = %T, want *ast.CallStmt", runForm.Body.Items[0])
	}
	if got := call.Callee.String(); got != "exec" {
		t.Fatalf("call callee = %q, want exec", got)
	}
}

func TestParseInvalidSource(t *testing.T) {
	src := []byte(`task build { deps = [1, 2 }`)
	_, diags := ParseFile(token.NewFileSet(), "broken.plano", src)
	if !diags.HasError() {
		t.Fatal("expected parse diagnostics")
	}
}
