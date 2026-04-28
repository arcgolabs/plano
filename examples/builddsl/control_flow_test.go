package builddsl_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLowerControlFlowSample(t *testing.T) {
	src, err := os.ReadFile("control_flow.plano")
	if err != nil {
		t.Fatal(err)
	}

	project := compileProject(t, src)
	assertDemoWorkspace(t, project)
	task := requireTask(t, project, "build")
	assertTaskOutputs(t, task, []string{filepath.Join("dist", "demo")})
	if got := task.Commands.Len(); got != 2 {
		t.Fatalf("commands = %d, want 2", got)
	}
	assertCommandArgs(t, task.Commands, 0, "./cmd/...")
	assertCommandArgs(t, task.Commands, 1, "./...")
}
