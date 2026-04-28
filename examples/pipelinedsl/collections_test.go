package pipelinedsl_test

import (
	"os"
	"testing"
)

func TestLowerCollectionsSample(t *testing.T) {
	src, err := os.ReadFile("collections.plano")
	if err != nil {
		t.Fatal(err)
	}

	project := compilePipeline(t, src)
	assertPipelineName(t, project, "matrix")
	assertStageOrder(t, project, []string{"lint", "unit", "package"})

	unit := requireStage(t, project, "unit")
	assertStageNeeds(t, unit, []string{"lint"})
	assertStageCommands(t, unit, 2, "./cmd/...")

	pkg := requireStage(t, project, "package")
	assertStageNeeds(t, pkg, []string{"lint", "unit"})
	assertStageCommands(t, pkg, 1, "./cmd/plano")
}
