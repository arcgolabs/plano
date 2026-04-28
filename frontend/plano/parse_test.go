package plano_test

import (
	"go/token"
	"testing"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/frontend/plano"
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

	file, diags := plano.ParseFile(token.NewFileSet(), "build.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	assertStatementCount(t, file, 3)
	assertConstDecl(t, file.Statements[0], "target")
	assertWorkspaceForm(t, file.Statements[1])
	assertTaskForm(t, file.Statements[2])
}

func TestParseInvalidSource(t *testing.T) {
	src := []byte(`task build { deps = [1, 2 }`)
	_, diags := plano.ParseFile(token.NewFileSet(), "broken.plano", src)
	if !diags.HasError() {
		t.Fatal("expected parse diagnostics")
	}
}

func TestParseElseIfAndLoopControl(t *testing.T) {
	src := []byte(`
task build {
  for idx, item in range(0, 3) {
    if idx == 1 {
      continue
    }
  }

  if true {
    run {
      exec("echo", "first")
    }
  } else if false {
    break
  } else {
    continue
  }
}
`)

	file, diags := plano.ParseFile(token.NewFileSet(), "flow.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	task := requireFormDecl(t, file.Statements[0])
	assertParsedIndexedLoop(t, task)
	assertParsedElseIfChain(t, task)
}

func assertParsedIndexedLoop(t *testing.T, task *ast.FormDecl) {
	t.Helper()
	loop, ok := task.Body.Items[0].(*ast.ForStmt)
	if !ok {
		t.Fatalf("item = %T, want *ast.ForStmt", task.Body.Items[0])
	}
	if loop.Index == nil || loop.Index.Name != "idx" || loop.Name == nil || loop.Name.Name != "item" {
		t.Fatalf("loop vars = %#v / %#v", loop.Index, loop.Name)
	}
}

func assertParsedElseIfChain(t *testing.T, task *ast.FormDecl) {
	t.Helper()
	stmt, ok := task.Body.Items[1].(*ast.IfStmt)
	if !ok {
		t.Fatalf("item = %T, want *ast.IfStmt", task.Body.Items[1])
	}
	if stmt.Else == nil || len(stmt.Else.Items) != 1 {
		t.Fatalf("else block = %#v", stmt.Else)
	}
	nested, ok := stmt.Else.Items[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("else item = %T, want nested *ast.IfStmt", stmt.Else.Items[0])
	}
	if nested.Else == nil || len(nested.Else.Items) != 1 {
		t.Fatalf("nested else block = %#v", nested.Else)
	}
	if _, ok := nested.Then.Items[0].(*ast.BreakStmt); !ok {
		t.Fatalf("nested then item = %T, want *ast.BreakStmt", nested.Then.Items[0])
	}
	if _, ok := nested.Else.Items[0].(*ast.ContinueStmt); !ok {
		t.Fatalf("nested else item = %T, want *ast.ContinueStmt", nested.Else.Items[0])
	}
}

func assertStatementCount(t *testing.T, file *ast.File, want int) {
	t.Helper()
	if got := len(file.Statements); got != want {
		t.Fatalf("expected %d statements, got %d", want, got)
	}
}

func assertConstDecl(t *testing.T, stmt ast.Stmt, want string) {
	t.Helper()
	constDecl, ok := stmt.(*ast.ConstDecl)
	if !ok {
		t.Fatalf("statement = %T, want *ast.ConstDecl", stmt)
	}
	if constDecl.Name.Name != want {
		t.Fatalf("const name = %q, want %q", constDecl.Name.Name, want)
	}
}

func assertWorkspaceForm(t *testing.T, stmt ast.Stmt) {
	t.Helper()
	workspace := requireFormDecl(t, stmt)
	if got := workspace.Head.String(); got != "workspace" {
		t.Fatalf("workspace head = %q, want workspace", got)
	}
	if workspace.Label != nil {
		t.Fatalf("workspace label = %#v, want nil", workspace.Label)
	}
	if got := len(workspace.Body.Items); got != 2 {
		t.Fatalf("workspace body items = %d, want 2", got)
	}
}

func assertTaskForm(t *testing.T, stmt ast.Stmt) {
	t.Helper()
	task := requireFormDecl(t, stmt)
	if got := task.Head.String(); got != "task" {
		t.Fatalf("task head = %q, want task", got)
	}
	if task.Label == nil || task.Label.Value != "build" {
		t.Fatalf("task label = %#v, want build", task.Label)
	}
	if got := len(task.Body.Items); got != 2 {
		t.Fatalf("task body items = %d, want 2", got)
	}
	runForm := requireFormDecl(t, task.Body.Items[1])
	if got := runForm.Head.String(); got != "run" {
		t.Fatalf("run form head = %q, want run", got)
	}
	call, ok := runForm.Body.Items[0].(*ast.CallStmt)
	if !ok {
		t.Fatalf("run body item = %T, want *ast.CallStmt", runForm.Body.Items[0])
	}
	if got := call.Callee.String(); got != "exec" {
		t.Fatalf("call callee = %q, want exec", got)
	}
}

func requireFormDecl(t *testing.T, node any) *ast.FormDecl {
	t.Helper()
	form, ok := node.(*ast.FormDecl)
	if !ok {
		t.Fatalf("node = %T, want *ast.FormDecl", node)
	}
	return form
}
