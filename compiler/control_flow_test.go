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

  if len(names) == 1 {
    outputs = [join_path("dist", "wrong")]
  } else if len(names) == 2 {
    outputs = [join_path("dist", "demo")]
  } else {
    outputs = [join_path("dist", "fallback")]
  }

  for pkg in packages {
    if pkg == "./..." {
      continue
    }
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
	assertTaskOutputs(t, doc.Forms[0], []string{filepath.Join("dist", "demo")})
	if got := len(doc.Forms[0].Forms); got != 2 {
		t.Fatalf("nested run forms = %d, want 2", got)
	}
	assertCallArgs(t, doc.Forms[0].Forms[0].Calls[0].Args, "./cmd/...")
	assertCallArgs(t, doc.Forms[0].Forms[1].Calls[0].Args, "./...")
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
