package compiler_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/diag"
)

func TestArtifactRejectsUnknownSchemaVersion(t *testing.T) {
	t.Parallel()

	var artifact compiler.Artifact
	err := json.Unmarshal([]byte(`{"schemaVersion":"plano.artifact/v999"}`), &artifact)
	if err == nil {
		t.Fatal("expected unknown schema version error")
	}
}

func TestArtifactPreservesRelatedDiagnostics(t *testing.T) {
	c := newTestCompiler(t)
	result := c.CompileStringDetailed(context.Background(), "duplicate.plano", `
const target: string = "dist"
const target: string = "release"
`)
	if !result.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
	artifact, err := result.Artifact()
	if err != nil {
		t.Fatal(err)
	}
	item, _ := artifact.Diagnostics.Get(0)
	if item.Code != diag.CodeDuplicateDefinition {
		t.Fatalf("artifact diagnostic code = %q", item.Code)
	}
	if item.Related.Len() != 1 {
		t.Fatalf("artifact related diagnostics = %d", item.Related.Len())
	}
	related, _ := item.Related.Get(0)
	if related.Message == "" || related.Span.Path != "duplicate.plano" {
		t.Fatalf("artifact related diagnostic = %#v", related)
	}

	roundTrip := decodeArtifactResult(t, artifact)
	if roundTrip.Diagnostics[0].Code != diag.CodeDuplicateDefinition {
		t.Fatalf("round-trip diagnostic code = %q", roundTrip.Diagnostics[0].Code)
	}
	if roundTrip.Diagnostics[0].Related.Len() != 1 {
		t.Fatalf("round-trip related diagnostics = %d", roundTrip.Diagnostics[0].Related.Len())
	}
}
