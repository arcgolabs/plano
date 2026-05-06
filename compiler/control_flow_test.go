package compiler_test

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCompileScriptControlFlowAndCollections(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  let packages = values({
    unit = "./...",
    integration = "./cmd/...",
    lint = "./internal/...",
  })
  let names = keys({
    unit = "./...",
    integration = "./cmd/...",
  })
  let suffix = len(names) == 2 ? "demo" : "fallback"

  if len(names) == 1 {
    outputs = [join_path("dist", "wrong")]
  } else if len(names) == 2 {
    outputs = [join_path("dist", suffix)]
  } else {
    outputs = [join_path("dist", "fallback")]
  }

  for pkg in packages where pkg != "./..." {
    if pkg == "./internal/..." {
      break
    }
    run {
      exec("go", "test", pkg)
    }
  }

  for idx in range(0, len(packages)) {
    if idx == 1 {
      break
    }
    run {
      exec("go", "test", packages[idx])
    }
  }
}
`)

	doc, diags := c.CompileSource(context.Background(), "flow.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	assertFormCount(t, doc, 1)
	build := formAt(t, doc.Forms, 0)
	assertTaskOutputs(t, build, []string{filepath.Join("dist", "demo")})
	if got := build.Forms.Len(); got != 2 {
		t.Fatalf("nested run forms = %d, want 2", got)
	}
	assertCallArgs(t, firstCall(t, nestedFormAt(t, build, 0)).Args, "./cmd/...")
	assertCallArgs(t, firstCall(t, nestedFormAt(t, build, 1)).Args, "./...")
}

func TestCompileRejectsNonBoolLoopWhere(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  for pkg in ["./..."] where pkg {
    run {
      exec("go", "test", pkg)
    }
  }
}
`)

	_, diags := c.CompileSource(context.Background(), "flow.plano", src)
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, diags, "for where clause expects bool, got string")
}

func TestCompileRejectsNonBoolConditionalCondition(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  outputs = [join_path("dist", "demo" ? "yes" : "no")]
}
`)

	_, diags := c.CompileSource(context.Background(), "conditional.plano", src)
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, diags, "conditional condition expects bool, got string")
}

func TestCompileRejectsLoopControlOutsideLoops(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  continue
}
`)

	_, diags := c.CompileSource(context.Background(), "flow.plano", src)
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, diags, "continue is only allowed inside loops")
}
