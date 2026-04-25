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

func assertContainsDiagnostic(t *testing.T, diags diag.Diagnostics, want string) {
	t.Helper()
	for _, item := range diags {
		if strings.Contains(item.Message, want) {
			return
		}
	}
	t.Fatalf("missing diagnostic containing %q", want)
}
