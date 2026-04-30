package compiler_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/arcgolabs/plano/compiler"
)

func TestCompileExprLangEvaluationUsesHostVarsFuncsAndLocals(t *testing.T) {
	c := newTestCompiler(t)
	registerExprVar(t, c, "branch", "main")
	registerExprFunc(t, c, "slug", func(params ...any) (any, error) {
		if len(params) != 1 {
			return nil, errors.New("slug expects one argument")
		}
		value, ok := params[0].(string)
		if !ok {
			return nil, errors.New("slug expects string")
		}
		return strings.ReplaceAll(value, "/", "-"), nil
	}, func(string) string { return "" })

	doc, diags := c.CompileSource(context.Background(), "expr.plano", []byte(`
task build {
  let prefix = "release"
  outputs = [expr("slug(prefix + '/' + branch)")]
}
`))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	assertFormCount(t, doc, 1)
	assertTaskOutputs(t, formAt(t, doc.Forms, 0), []string{"release-main"})
}

func TestCompileExprLangEvaluationUsesOverrideMap(t *testing.T) {
	c := newTestCompiler(t)
	registerExprVar(t, c, "branch", "main")

	doc, diags := c.CompileSource(context.Background(), "expr.plano", []byte(`
task build {
  outputs = [expr_eval("dir + '/' + branch", {
    dir = "dist",
    branch = "override",
  })]
}
`))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	assertFormCount(t, doc, 1)
	assertTaskOutputs(t, formAt(t, doc.Forms, 0), []string{"dist/override"})
}

func TestCompileExprLangEvaluationReportsCompileErrors(t *testing.T) {
	c := newTestCompiler(t)
	_, diags := c.CompileSource(context.Background(), "expr.plano", []byte(`
task build {
  outputs = [expr("missing + 1")]
}
`))
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, diags, `unknown name missing`)
}

func registerExprVar(t *testing.T, c *compiler.Compiler, name string, value any) {
	t.Helper()
	if err := c.RegisterExprVar(name, value); err != nil {
		t.Fatal(err)
	}
}

func registerExprFunc(
	t *testing.T,
	c *compiler.Compiler,
	name string,
	fn func(params ...any) (any, error),
	types ...any,
) {
	t.Helper()
	if err := c.RegisterExprFunc(name, fn, types...); err != nil {
		t.Fatal(err)
	}
}
