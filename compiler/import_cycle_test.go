package compiler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCompileFileWithImportCyclePathDiagnostic(t *testing.T) {
	c := newTestCompiler(t)
	dir := t.TempDir()

	root := filepath.Join(dir, "build.plano")
	child := filepath.Join(dir, "tasks.plano")

	if err := os.WriteFile(child, []byte(`
import "build.plano"

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

	_, diags := c.CompileFile(context.Background(), root)
	if !diags.HasError() {
		t.Fatal("expected import cycle diagnostics")
	}
	want := "import cycle detected: " + root + " -> " + child + " -> " + root
	found := false
	for _, item := range diags {
		if item.Message == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("diagnostics = %#v, want %q", diags, want)
	}
}
