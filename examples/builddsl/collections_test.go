package builddsl_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLowerCollectionsSample(t *testing.T) {
	src, err := os.ReadFile("collections.plano")
	if err != nil {
		t.Fatal(err)
	}

	project := compileProject(t, src)
	assertDemoWorkspace(t, project)
	task := requireTask(t, project, "build")
	assertTaskOutputs(t, task, []string{
		filepath.Join("dist", "demo"),
		filepath.Join("dist", "backup"),
	})
	if got := len(task.Commands); got != 1 {
		t.Fatalf("commands = %d, want 1", got)
	}
	if task.Commands[0].Args[1] != filepath.Join("dist", "demo") {
		t.Fatalf("command args = %#v", task.Commands[0].Args)
	}
}
