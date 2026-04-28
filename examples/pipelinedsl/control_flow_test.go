package pipelinedsl_test

import (
	"os"
	"testing"
)

func TestLowerControlFlowSample(t *testing.T) {
	src, err := os.ReadFile("control_flow.plano")
	if err != nil {
		t.Fatal(err)
	}

	project := compilePipeline(t, src)
	assertPipelineName(t, project, "flow")
	assertStageOrder(t, project, []string{"lint", "test"})
	stage := requireStage(t, project, "test")
	assertStageCommands(t, stage, 2, "./...")
}
