//nolint:testpackage,cyclop,gocognit,gocyclo // Example DSL tests stay in-package for concise helper access.
package builddsl

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/plano/compiler"
)

func TestLowerProject(t *testing.T) {
	c := compiler.New(compiler.Options{
		LookupEnv: func(string) (string, bool) { return "", false },
	})
	if err := Register(c); err != nil {
		t.Fatal(err)
	}

	src := []byte(`
fn output(name: string): path {
  return join_path("dist", name)
}

workspace {
  name = "demo"
  default = build
}

task prepare {
  run {
    exec("mkdir", "-p", "dist")
  }
}

task build {
  deps = [prepare]
  outputs = [output("demo")]

  for pkg in ["./...", "./cmd/..."] {
    run {
      exec("go", "test", pkg)
    }
  }
}
`)

	doc, diags := c.CompileSource(context.Background(), "build.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	project, err := Lower(doc)
	if err != nil {
		t.Fatal(err)
	}

	workspace, ok := project.Workspace.Get()
	if !ok {
		t.Fatal("expected workspace")
	}
	if workspace.Name != "demo" || workspace.DefaultTask != "build" {
		t.Fatalf("workspace = %#v", workspace)
	}

	if got := project.Tasks.Keys(); len(got) != 2 || got[0] != "prepare" || got[1] != "build" {
		t.Fatalf("task order = %#v, want [prepare build]", got)
	}

	build := project.Tasks.GetOption("build")
	if build.IsAbsent() {
		t.Fatal("expected build task")
	}
	task := build.MustGet()
	if got := task.Deps; len(got) != 1 || got[0] != "prepare" {
		t.Fatalf("deps = %#v, want [prepare]", got)
	}
	if got := task.Outputs; len(got) != 1 || got[0] != filepath.Join("dist", "demo") {
		t.Fatalf("outputs = %#v", got)
	}
	if got := len(task.Commands); got != 2 {
		t.Fatalf("commands = %d, want 2", got)
	}
	if got := task.Commands[0].Args[2]; got != "./..." {
		t.Fatalf("first command arg = %#v, want ./...", got)
	}
	if got := task.Commands[1].Args[2]; got != "./cmd/..." {
		t.Fatalf("second command arg = %#v, want ./cmd/...", got)
	}
}

func TestLowerGoPluginForms(t *testing.T) {
	c := compiler.New(compiler.Options{
		LookupEnv: func(string) (string, bool) { return "", false },
	})
	if err := Register(c); err != nil {
		t.Fatal(err)
	}

	src := []byte(`
workspace {
  name = "demo"
  default = build
}

task prepare {}

go.test test {
  deps = [prepare]
  packages = ["./...", "./cmd/..."]
}

go.binary build {
  deps = [prepare, test]
  main = "./cmd/demo"
  out = "dist/demo"
}
`)

	doc, diags := c.CompileSource(context.Background(), "build.plano", src)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	project, err := Lower(doc)
	if err != nil {
		t.Fatal(err)
	}

	testTask, ok := project.Tasks.Get("test")
	if !ok {
		t.Fatal("expected test task")
	}
	if got := testTask.Commands[0].Args; len(got) != 4 || got[0] != "go" || got[1] != "test" {
		t.Fatalf("go.test command = %#v", got)
	}

	buildTask, ok := project.Tasks.Get("build")
	if !ok {
		t.Fatal("expected build task")
	}
	if got := buildTask.Deps; len(got) != 2 || got[0] != "prepare" || got[1] != "test" {
		t.Fatalf("go.binary deps = %#v", got)
	}
	if got := buildTask.Commands[0].Args; len(got) != 5 || got[0] != "go" || got[1] != "build" || got[3] != "dist/demo" {
		t.Fatalf("go.binary command = %#v", got)
	}
}
