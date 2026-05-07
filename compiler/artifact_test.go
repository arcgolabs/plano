package compiler_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

func TestCompileStringArtifactRoundTrip(t *testing.T) {
	c := newTestCompiler(t)
	artifact, err := c.CompileStringArtifact(context.Background(), "artifact.plano", artifactFixtureSource())
	if err != nil {
		t.Fatal(err)
	}
	assertArtifactShape(t, artifact)
	result := decodeArtifactResult(t, artifact)
	assertRoundTripResult(t, result)
}

func TestArtifactPreservesDiagnostics(t *testing.T) {
	c := newTestCompiler(t)
	result := c.CompileStringDetailed(context.Background(), "invalid.plano", `
workspace {
  name = 1
  default = build
}
`)
	if !result.Diagnostics.HasError() {
		t.Fatal("expected diagnostics")
	}
	artifact, err := result.Artifact()
	if err != nil {
		t.Fatal(err)
	}
	assertArtifactDiagnostic(t, artifact)
	assertRoundTripDiagnostic(t, decodeArtifactResult(t, artifact))
}

func TestArtifactTypeAndValueRoundTrip(t *testing.T) {
	t.Parallel()

	assertArtifactTypeRoundTrip(t)
	assertArtifactValueRoundTrip(t)
}

func artifactFixtureSource() string {
	return `
const target: string = "dist/demo"
const meta: map<string> = {
  name = "demo",
  env = "dev",
}

fn output(name: string): path {
  return join_path("dist", name)
}

workspace {
  name = "demo"
  default = build
}

task build {
  deps = [prepare]
  outputs = [output("app")]

  run {
    exec("go", "test", "./...")
  }
}

task prepare {}
`
}

func assertArtifactShape(t *testing.T, artifact *compiler.Artifact) {
	t.Helper()
	if artifact == nil {
		t.Fatal("expected artifact")
	}
	assertArtifactDiagnosticsEmpty(t, artifact)
	assertArtifactSections(t, artifact)
	assertArtifactWorkspaceSpan(t, artifact)
	assertArtifactBindingState(t, artifact)
	assertArtifactHIRState(t, artifact)
}

func assertArtifactDiagnosticsEmpty(t *testing.T, artifact *compiler.Artifact) {
	t.Helper()
	if got := artifact.Diagnostics.Len(); got != 0 {
		t.Fatalf("artifact diagnostics = %d, want 0", got)
	}
}

func assertArtifactSections(t *testing.T, artifact *compiler.Artifact) {
	t.Helper()
	if artifact.SchemaVersion != compiler.ArtifactSchemaVersion {
		t.Fatalf("artifact schema version = %q, want %q", artifact.SchemaVersion, compiler.ArtifactSchemaVersion)
	}
	if artifact.Document == nil || artifact.Binding == nil || artifact.Checks == nil || artifact.HIR == nil {
		t.Fatalf("artifact sections missing: %#v", artifact)
	}
	if got := artifact.Document.Forms.Len(); got != 3 {
		t.Fatalf("artifact forms = %d, want 3", got)
	}
}

func assertArtifactWorkspaceSpan(t *testing.T, artifact *compiler.Artifact) {
	t.Helper()
	workspace, _ := artifact.Document.Forms.Get(0)
	if workspace.Span.Path != "artifact.plano" || workspace.Span.Start.Line == 0 {
		t.Fatalf("workspace span = %#v", workspace.Span)
	}
}

func assertArtifactBindingState(t *testing.T, artifact *compiler.Artifact) {
	t.Helper()
	if got := artifact.Binding.Files.Len(); got != 1 {
		t.Fatalf("binding files = %d, want 1", got)
	}
	if got, _ := artifact.Binding.Files.Get(0); got != "artifact.plano" {
		t.Fatalf("binding file = %q, want artifact.plano", got)
	}
	if _, ok := artifact.Binding.Symbols.Get("build"); !ok {
		t.Fatal("expected build symbol in artifact binding")
	}
}

func assertArtifactHIRState(t *testing.T, artifact *compiler.Artifact) {
	t.Helper()
	if _, ok := artifact.HIR.Consts.Get("meta"); !ok {
		t.Fatal("expected meta const in artifact HIR")
	}
}

func decodeArtifactResult(t *testing.T, artifact *compiler.Artifact) compiler.Result {
	t.Helper()
	data, err := artifact.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var decoded compiler.Artifact
	decodeErr := decoded.UnmarshalBinary(data)
	if decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if decoded.SchemaVersion != compiler.ArtifactSchemaVersion {
		t.Fatalf("decoded artifact schema version = %q, want %q", decoded.SchemaVersion, compiler.ArtifactSchemaVersion)
	}
	result, err := decoded.Result()
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertRoundTripResult(t *testing.T, result compiler.Result) {
	t.Helper()
	if result.FileSet != nil {
		t.Fatal("expected round-tripped result to omit fileset")
	}
	if got := len(result.Diagnostics); got != 0 {
		t.Fatalf("round-tripped diagnostics = %d, want 0", got)
	}
	assertFormCount(t, result.Document, 3)
	assertWorkspaceDefault(t, formAt(t, result.Document.Forms, 0), "build")
	assertTaskOutputs(t, formAt(t, result.Document.Forms, 1), []string{filepath.Join("dist", "app")})
	if got := result.Binding.Functions.Len(); got != 1 {
		t.Fatalf("round-tripped functions = %d, want 1", got)
	}
	if got := result.Checks.Calls.Len(); got == 0 {
		t.Fatal("expected round-tripped call checks")
	}
	metaConst, ok := result.HIR.Consts.Get("meta")
	if !ok {
		t.Fatal("missing round-tripped HIR const")
	}
	meta, ok := metaConst.Value.(*mapping.OrderedMap[string, any])
	if !ok {
		t.Fatalf("meta const value = %T, want ordered map", metaConst.Value)
	}
	if got, _ := meta.Get("env"); got != "dev" {
		t.Fatalf("meta env = %#v, want dev", got)
	}
}

func assertArtifactDiagnostic(t *testing.T, artifact *compiler.Artifact) {
	t.Helper()
	if got := artifact.Diagnostics.Len(); got == 0 {
		t.Fatal("expected artifact diagnostics")
	}
	item, _ := artifact.Diagnostics.Get(0)
	if item.Code != diag.CodeTypeMismatch || item.Message == "" || item.Span.Path != "invalid.plano" || item.Span.Start.Line == 0 {
		t.Fatalf("artifact diagnostic = %#v", item)
	}
}

func assertRoundTripDiagnostic(t *testing.T, roundTrip compiler.Result) {
	t.Helper()
	if !roundTrip.Diagnostics.HasError() {
		t.Fatal("expected round-tripped diagnostics")
	}
	if roundTrip.Diagnostics[0].Code != diag.CodeTypeMismatch {
		t.Fatalf("round-trip diagnostic code = %q", roundTrip.Diagnostics[0].Code)
	}
	if roundTrip.Diagnostics[0].Pos.IsValid() || roundTrip.Diagnostics[0].End.IsValid() {
		t.Fatalf("expected zeroed diagnostic positions, got %#v", roundTrip.Diagnostics[0])
	}
}

func assertArtifactTypeRoundTrip(t *testing.T) {
	t.Helper()
	typ, err := (compiler.ArtifactType{
		Kind: "map",
		Elem: &compiler.ArtifactType{
			Kind: "list",
			Elem: &compiler.ArtifactType{Kind: "ref", Name: "task"},
		},
	}).Type()
	if err != nil {
		t.Fatal(err)
	}
	if got := typ.String(); got != "map<list<ref<task>>>" {
		t.Fatalf("type string = %q", got)
	}
}

func assertArtifactValueRoundTrip(t *testing.T) {
	t.Helper()
	value, err := (compiler.ArtifactValue{
		Kind: "map",
		Fields: orderedArtifactFields(
			artifactField{
				Name:  "task",
				Value: compiler.ArtifactValue{Kind: "ref", Ref: &schema.Ref{Kind: "task", Name: "build"}},
			},
			artifactField{
				Name: "labels",
				Value: compiler.ArtifactValue{
					Kind: "list",
					Items: artifactValues(
						compiler.ArtifactValue{Kind: "string", String: "ci"},
						compiler.ArtifactValue{Kind: "string", String: "release"},
					),
				},
			},
		),
	}).Value()
	if err != nil {
		t.Fatal(err)
	}
	fields, ok := value.(*mapping.OrderedMap[string, any])
	if !ok {
		t.Fatalf("artifact value = %T, want ordered map", value)
	}
	task, _ := fields.Get("task")
	if task != (schema.Ref{Kind: "task", Name: "build"}) {
		t.Fatalf("task ref = %#v", task)
	}
	labels, _ := fields.Get("labels")
	items, ok := labels.([]any)
	if !ok || len(items) != 2 || items[1] != "release" {
		t.Fatalf("labels = %#v", labels)
	}
}

func artifactValues(items ...compiler.ArtifactValue) list.List[compiler.ArtifactValue] {
	values := list.NewListWithCapacity[compiler.ArtifactValue](len(items))
	for index := range items {
		values.Add(items[index])
	}
	return *values
}

type artifactField struct {
	Name  string
	Value compiler.ArtifactValue
}

func orderedArtifactFields(items ...artifactField) *mapping.OrderedMap[string, compiler.ArtifactValue] {
	fields := mapping.NewOrderedMapWithCapacity[string, compiler.ArtifactValue](len(items))
	for index := range items {
		fields.Set(items[index].Name, items[index].Value)
	}
	return fields
}
