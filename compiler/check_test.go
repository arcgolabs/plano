package compiler_test

import (
	"context"
	"strings"
	"testing"

	"github.com/arcgolabs/plano/diag"
)

func TestCheckSourceDetailed(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
const target: string = "dist/demo"

fn output(name: string): path {
  return name
}

task build {
  outputs = [target, output("demo")]
  run {
    exec("go", "test", "./...")
  }
}
`)

	result := c.CheckSourceDetailed(context.Background(), "build.plano", src)
	if result.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if result.Checks == nil {
		t.Fatal("expected checks")
	}
	if got := result.Checks.Fields.Len(); got != 1 {
		t.Fatalf("field checks = %d", got)
	}
	field := result.Checks.Fields.Values()[0]
	if field.Field != "outputs" {
		t.Fatalf("field = %q", field.Field)
	}
	if got := field.Expected.String(); got != "list<path>" {
		t.Fatalf("expected type = %q", got)
	}
	if got := field.Actual.String(); got != "list<string>" {
		t.Fatalf("actual type = %q", got)
	}
	if got := result.Checks.Calls.Len(); got != 2 {
		t.Fatalf("call checks = %d", got)
	}
}

func TestCheckSourceDetailedReportsTypeErrors(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
fn output(name: string): path {
  return 1
}

task build {
  if 1 {
    outputs = ["dist/demo"]
  }
  run {
    exec(1)
  }
}
`)

	result := c.CheckSourceDetailed(context.Background(), "build.plano", src)
	if !result.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, result.Diagnostics, "return expects path, got int")
	assertContainsDiagnostic(t, result.Diagnostics, "if condition expects bool, got int")
	assertContainsDiagnostic(t, result.Diagnostics, "action argument 1 expects string, got int")
}

func TestCheckSourceDetailedReportsLoopControlErrors(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  break
}

fn pick(): int {
  continue
  return 1
}
`)

	result := c.CheckSourceDetailed(context.Background(), "flow.plano", src)
	if !result.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, result.Diagnostics, "break is only allowed inside loops")
	assertContainsDiagnostic(t, result.Diagnostics, "continue is only allowed inside loops")
}

func TestCheckSourceDetailedReportsAssignmentErrors(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
fn output(): string {
  const name = "demo"
  name = "next"
  missing = "x"
  return name
}

task build {
  let enabled = true
  enabled = 1
  outputs = [output()]
}
`)

	result := c.CheckSourceDetailed(context.Background(), "assign.plano", src)
	if !result.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, result.Diagnostics, `cannot assign to const "name"`)
	assertContainsDiagnostic(t, result.Diagnostics, `undefined local binding "missing"`)
	assertContainsDiagnostic(t, result.Diagnostics, "assignment \"enabled\" expects bool, got int")
}

func TestCheckSourceDetailedReportsCollectionBuiltinErrors(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  let items = ["one", "two"]
  let data = {name = "demo"}
  let a = get(items, "x")
  let b = get(data, 1)
  let c = slice(data, 0)
  let d = has(data, 1)
  outputs = [a, b, c, d]
}
`)

	result := c.CheckSourceDetailed(context.Background(), "collections.plano", src)
	if !result.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, result.Diagnostics, "get index expects int, got string")
	assertContainsDiagnostic(t, result.Diagnostics, "get key expects string, got int")
	assertContainsDiagnostic(t, result.Diagnostics, "slice expects list argument")
	assertContainsDiagnostic(t, result.Diagnostics, "has key expects string, got int")
}

func TestCheckSourceDetailedAddsDiagnosticSuggestions(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
workspace {
  name = "demo"
  default = buld
}

task build {
  outputz = []
  outputs = [join_pat("dist", "demo")]
  rn {
    exec("echo")
  }
  run {
    exce("go", "test")
  }
}
`)

	result := c.CheckSourceDetailed(context.Background(), "suggestions.plano", src)
	assertDiagnosticSuggestion(t, result.Diagnostics, diag.CodeUndefinedName, "build")
	assertDiagnosticSuggestion(t, result.Diagnostics, diag.CodeUnknownField, "outputs")
	assertDiagnosticSuggestion(t, result.Diagnostics, diag.CodeUnknownNestedForm, "run")
	assertDiagnosticSuggestion(t, result.Diagnostics, diag.CodeUnknownFunction, "join_path")
	assertDiagnosticSuggestion(t, result.Diagnostics, diag.CodeUnknownAction, "exec")
}

func assertContainsDiagnostic(t *testing.T, diags diag.Diagnostics, want string) {
	t.Helper()
	for index := range diags {
		item := diags[index]
		if strings.Contains(item.Message, want) {
			return
		}
	}
	t.Fatalf("missing diagnostic containing %q", want)
}

func assertDiagnosticSuggestion(t *testing.T, diags diag.Diagnostics, code diag.Code, replacement string) {
	t.Helper()
	for index := range diags {
		item := diags[index]
		if item.Code != code {
			continue
		}
		for suggestionIndex := range item.Suggestions.Len() {
			suggestion, _ := item.Suggestions.Get(suggestionIndex)
			if suggestion.Replacement == replacement {
				return
			}
		}
		t.Fatalf("diagnostic %q suggestions = %#v, missing %q", code, item.Suggestions, replacement)
	}
	t.Fatalf("missing diagnostic code %q", code)
}
