package pipelinedsl_test

import (
	"context"
	"os"
	"testing"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/examples/pipelinedsl"
)

func TestLowerSample(t *testing.T) {
	src := mustReadSample(t)
	project := compilePipeline(t, src)
	assertPipelineName(t, project, "release")
	assertStageOrder(t, project, []string{"lint", "test", "build"})
	stage := requireStage(t, project, "test")
	assertStageCommands(t, stage, 2, "./cmd/...")
}

func mustReadSample(t *testing.T) []byte {
	t.Helper()
	src, err := os.ReadFile("sample.plano")
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func compilePipeline(t *testing.T, src []byte) *pipelinedsl.Pipeline {
	t.Helper()
	c := compiler.New(compiler.Options{})
	if err := pipelinedsl.Register(c); err != nil {
		t.Fatal(err)
	}
	result := c.CompileSourceDetailed(context.Background(), "sample.plano", src)
	if result.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	if result.HIR == nil {
		t.Fatal("expected HIR")
	}
	project, err := pipelinedsl.Lower(result.HIR)
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func assertPipelineName(t *testing.T, project *pipelinedsl.Pipeline, want string) {
	t.Helper()
	if project.Name != want {
		t.Fatalf("name = %q", project.Name)
	}
}

func assertStageOrder(t *testing.T, project *pipelinedsl.Pipeline, want []string) {
	t.Helper()
	got := project.Stages.Keys()
	if len(got) != len(want) {
		t.Fatalf("stage order = %#v, want %#v", got, want)
	}
	for idx, item := range want {
		if got[idx] != item {
			t.Fatalf("stage order = %#v, want %#v", got, want)
		}
	}
}

func requireStage(t *testing.T, project *pipelinedsl.Pipeline, name string) pipelinedsl.Stage {
	t.Helper()
	stage, ok := project.Stages.Get(name)
	if !ok {
		t.Fatalf("expected %s stage", name)
	}
	return stage
}

func assertStageCommands(t *testing.T, stage pipelinedsl.Stage, wantCount int, wantLast string) {
	t.Helper()
	if stage.Commands.Len() != wantCount {
		t.Fatalf("commands = %d", stage.Commands.Len())
	}
	command, _ := stage.Commands.Get(wantCount - 1)
	args := command.Args.Values()
	if got := args[2]; got != wantLast {
		t.Fatalf("last command = %#v", args)
	}
}

func assertStageNeeds(t *testing.T, stage pipelinedsl.Stage, want []string) {
	t.Helper()
	got := stage.Needs.Values()
	if len(got) != len(want) {
		t.Fatalf("needs = %#v, want %#v", got, want)
	}
	for idx, item := range want {
		if got[idx] != item {
			t.Fatalf("needs = %#v, want %#v", got, want)
		}
	}
}
