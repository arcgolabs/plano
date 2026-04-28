package builddsl_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/examples/builddsl"
)

func TestLowerProject(t *testing.T) {
	project := compileProject(t, []byte(`
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
`))

	assertDemoWorkspace(t, project)
	assertTaskOrder(t, project, []string{"prepare", "build"})
	task := requireTask(t, project, "build")
	assertTaskDeps(t, task, []string{"prepare"})
	assertTaskOutputs(t, task, []string{filepath.Join("dist", "demo")})
	assertCommandArgs(t, task.Commands, 0, "./...")
	assertCommandArgs(t, task.Commands, 1, "./cmd/...")
}

func TestLowerGoPluginForms(t *testing.T) {
	project := compileProject(t, []byte(`
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
`))

	testTask := requireTask(t, project, "test")
	assertGoTestCommand(t, testTask.Commands)
	buildTask := requireTask(t, project, "build")
	assertTaskDeps(t, buildTask, []string{"prepare", "test"})
	assertGoBuildCommand(t, buildTask.Commands)
}

func TestLowerSample(t *testing.T) {
	src := mustReadBuildSample(t)
	project := compileProject(t, src)
	assertDemoWorkspace(t, project)
	assertTaskOrder(t, project, []string{"prepare", "test", "build"})
	buildTask := requireTask(t, project, "build")
	assertTaskDeps(t, buildTask, []string{"prepare", "test"})
	assertTaskOutputs(t, buildTask, []string{filepath.Join("dist", "plano")})
}

func mustReadBuildSample(t *testing.T) []byte {
	t.Helper()
	src, err := os.ReadFile("sample.plano")
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func compileProject(t *testing.T, src []byte) *builddsl.Project {
	t.Helper()

	c := compiler.New(compiler.Options{
		LookupEnv: func(string) (string, bool) { return "", false },
	})
	if err := builddsl.Register(c); err != nil {
		t.Fatal(err)
	}

	result := c.CompileSourceDetailed(context.Background(), "build.plano", src)
	if result.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if result.Document == nil || result.HIR == nil {
		t.Fatal("expected compile outputs")
	}
	project, err := builddsl.Lower(result.HIR)
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func assertDemoWorkspace(t *testing.T, project *builddsl.Project) {
	t.Helper()
	workspace, ok := project.Workspace.Get()
	if !ok {
		t.Fatal("expected workspace")
	}
	if workspace.Name != "demo" || workspace.DefaultTask != "build" {
		t.Fatalf("workspace = %#v", workspace)
	}
}

func assertTaskOrder(t *testing.T, project *builddsl.Project, want []string) {
	t.Helper()
	got := project.Tasks.Keys()
	if len(got) != len(want) {
		t.Fatalf("task order = %#v, want %#v", got, want)
	}
	for idx, item := range want {
		if got[idx] != item {
			t.Fatalf("task order = %#v, want %#v", got, want)
		}
	}
}

func requireTask(t *testing.T, project *builddsl.Project, name string) builddsl.Task {
	t.Helper()
	task, ok := project.Tasks.Get(name)
	if !ok {
		t.Fatalf("expected %s task", name)
	}
	return task
}

func assertTaskDeps(t *testing.T, task builddsl.Task, want []string) {
	t.Helper()
	got := task.Deps.Values()
	if len(got) != len(want) {
		t.Fatalf("deps = %#v, want %#v", task.Deps, want)
	}
	for idx, item := range want {
		if got[idx] != item {
			t.Fatalf("deps = %#v, want %#v", task.Deps, want)
		}
	}
}

func assertTaskOutputs(t *testing.T, task builddsl.Task, want []string) {
	t.Helper()
	got := task.Outputs.Values()
	if len(got) != len(want) {
		t.Fatalf("outputs = %#v, want %#v", task.Outputs, want)
	}
	for idx, item := range want {
		if got[idx] != item {
			t.Fatalf("outputs = %#v, want %#v", task.Outputs, want)
		}
	}
}

func assertCommandArgs(t *testing.T, commands list.List[builddsl.Command], index int, wantLast string) {
	t.Helper()
	if commands.Len() <= index {
		t.Fatalf("commands = %#v", commands)
	}
	command, _ := commands.Get(index)
	args := command.Args.Values()
	if got := args[2]; got != wantLast {
		t.Fatalf("command args = %#v, want last %#v", args, wantLast)
	}
}

func assertGoTestCommand(t *testing.T, commands list.List[builddsl.Command]) {
	t.Helper()
	if commands.Len() == 0 {
		t.Fatal("expected commands")
	}
	command, _ := commands.Get(0)
	args := command.Args.Values()
	if len(args) != 4 || args[0] != "go" || args[1] != "test" {
		t.Fatalf("go.test command = %#v", args)
	}
}

func assertGoBuildCommand(t *testing.T, commands list.List[builddsl.Command]) {
	t.Helper()
	if commands.Len() == 0 {
		t.Fatal("expected commands")
	}
	command, _ := commands.Get(0)
	args := command.Args.Values()
	if len(args) != 5 || args[0] != "go" || args[1] != "build" || args[3] != "dist/demo" {
		t.Fatalf("go.binary command = %#v", args)
	}
}
