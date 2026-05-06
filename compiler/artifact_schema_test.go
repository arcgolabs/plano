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

func TestArtifactAcceptsPreviousSchemaVersion(t *testing.T) {
	t.Parallel()

	var artifact compiler.Artifact
	err := json.Unmarshal([]byte(`{"schemaVersion":"plano.artifact/v1"}`), &artifact)
	if err != nil {
		t.Fatalf("unexpected v1 artifact error: %v", err)
	}
	if artifact.SchemaVersion != "plano.artifact/v1" {
		t.Fatalf("schema version = %q", artifact.SchemaVersion)
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

func TestArtifactPreservesDiagnosticSuggestions(t *testing.T) {
	c := newTestCompiler(t)
	result := c.CompileStringDetailed(context.Background(), "suggestions.plano", `
task build {
  outputs = [join_pat("dist", "demo")]
}
`)
	if !result.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
	artifact, err := result.Artifact()
	if err != nil {
		t.Fatal(err)
	}
	item := artifactDiagnosticWithCode(t, artifact, diag.CodeUnknownFunction)
	if item.Suggestions.Len() == 0 {
		t.Fatal("expected artifact suggestions")
	}
	suggestion, _ := item.Suggestions.Get(0)
	if suggestion.Replacement != "join_path" || suggestion.Span.Path != "suggestions.plano" {
		t.Fatalf("artifact suggestion = %#v", suggestion)
	}

	roundTrip := decodeArtifactResult(t, artifact)
	roundTripItem := diagnosticWithCode(t, roundTrip.Diagnostics, diag.CodeUnknownFunction)
	roundTripSuggestion, _ := roundTripItem.Suggestions.Get(0)
	if roundTripSuggestion.Title != `Replace with "join_path"` || roundTripSuggestion.Replacement != "join_path" {
		t.Fatalf("round-trip suggestion = %#v", roundTripSuggestion)
	}
}

func artifactDiagnosticWithCode(t *testing.T, artifact *compiler.Artifact, code diag.Code) compiler.ArtifactDiagnostic {
	t.Helper()
	for index := range artifact.Diagnostics.Len() {
		item, _ := artifact.Diagnostics.Get(index)
		if item.Code == code {
			return item
		}
	}
	t.Fatalf("artifact diagnostics = %#v, missing %q", artifact.Diagnostics, code)
	return compiler.ArtifactDiagnostic{}
}

func diagnosticWithCode(t *testing.T, items diag.Diagnostics, code diag.Code) diag.Diagnostic {
	t.Helper()
	for index := range items {
		item := items[index]
		if item.Code == code {
			return item
		}
	}
	t.Fatalf("diagnostics = %#v, missing %q", items, code)
	return diag.Diagnostic{}
}
