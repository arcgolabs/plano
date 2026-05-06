package compiler_test

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCompileCollectionBuiltinsAndIndexedLoops(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
fn packages(): list<string> {
  let base = append(["./..."], "./cmd/...")
  if "cli" in merge({unit = "./..."}, {cli = "./cmd/..."}) {
    base = concat(base, ["./internal/..."])
  }
  return base
}

task build {
  let outputs_map = merge(
    {main = join_path("dist", "demo")},
    {backup = join_path("dist", "backup")},
  )
  outputs = values(outputs_map)

  for idx, pkg in packages() where idx != 1 {
    if pkg in ["./internal/..."] {
      break
    }
    run {
      exec("go", "test", pkg)
    }
  }

  for name, output in outputs_map {
    if name == "backup" {
      break
    }
    run {
      exec("echo", output)
    }
  }
}
`)

	doc, diags := c.CompileSource(context.Background(), "collections.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	assertFormCount(t, doc, 1)
	build := formAt(t, doc.Forms, 0)
	assertTaskOutputs(t, build, []string{
		filepath.Join("dist", "demo"),
		filepath.Join("dist", "backup"),
	})
	if got := build.Forms.Len(); got != 2 {
		t.Fatalf("nested run forms = %d, want 2", got)
	}
	first := firstCall(t, nestedFormAt(t, build, 0))
	firstArgs := first.Args.Values()
	if first.Name != "exec" || len(firstArgs) != 3 || firstArgs[2] != "./..." {
		t.Fatalf("first call = %#v", first)
	}
	second := firstCall(t, nestedFormAt(t, build, 1))
	secondArgs := second.Args.Values()
	if second.Name != "exec" || len(secondArgs) != 2 || secondArgs[1] != filepath.Join("dist", "demo") {
		t.Fatalf("second call = %#v", second)
	}
}

func TestCompileGetAndSliceBuiltins(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  let items = append(["one"], "two", "three")
  let names = slice(items, 1, len(items))
  let primary = get({name = join_path("dist", "demo")}, "name", "dist/fallback")
  let missing = get({name = join_path("dist", "demo")}, "missing", "dist/fallback")
  outputs = concat([primary], [missing], names)
}
`)

	doc, diags := c.CompileSource(context.Background(), "access.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	assertFormCount(t, doc, 1)
	assertTaskOutputs(t, formAt(t, doc.Forms, 0), []string{
		filepath.Join("dist", "demo"),
		"dist/fallback",
		"two",
		"three",
	})
}

func TestCompileRejectsInvalidMembershipExpressions(t *testing.T) {
	c := newTestCompiler(t)
	src := []byte(`
task build {
  if 1 in {name = "demo"} {
    outputs = ["wrong"]
  }
  if "demo" in "demo" {
    outputs = ["wrong"]
  }
}
`)

	_, diags := c.CompileSource(context.Background(), "membership.plano", src)
	if !diags.HasError() {
		t.Fatal("expected diagnostics")
	}
	assertContainsDiagnostic(t, diags, "operator in map key expects string, got int")
	assertContainsDiagnostic(t, diags, "operator in expects list or map")
}
