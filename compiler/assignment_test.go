package compiler_test

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCompileLocalReassignment(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
fn output(name: string): path {
  let result = join_path("dist", "tmp")
  result = join_path("dist", name)
  return result
}

task build {
  let selected = "dist/tmp"
  selected = output("demo")
  outputs = [selected]
}
`)

	doc, diags := c.CompileSource(context.Background(), "assign.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	assertFormCount(t, doc, 1)
	assertTaskOutputs(t, doc.Forms[0], []string{filepath.Join("dist", "demo")})
}

func TestCompileRejectsFieldShadowingBindings(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  let outputs = ["dist/demo"]
}
`)

	_, diags := c.CompileSource(context.Background(), "assign.plano", src)
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, diags, `binding "outputs" conflicts with field "outputs" in task`)
}
