package compiler_test

import (
	"context"
	"errors"
	"testing"

	compilerpkg "github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/examples/builddsl"
)

func BenchmarkBindStringDetailed(b *testing.B) {
	compiler := newRegisteredCompiler(b)
	src := artifactFixtureSource()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := compiler.BindStringDetailed(ctx, "bench.plano", src)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkCheckStringDetailed(b *testing.B) {
	compiler := newRegisteredCompiler(b)
	src := artifactFixtureSource()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := compiler.CheckStringDetailed(ctx, "bench.plano", src)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkCompileStringDetailed(b *testing.B) {
	compiler := newRegisteredCompiler(b)
	src := artifactFixtureSource()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := compiler.CompileStringDetailed(ctx, "bench.plano", src)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkCompileStringArtifact(b *testing.B) {
	compiler := newRegisteredCompiler(b)
	src := artifactFixtureSource()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		artifact, err := compiler.CompileStringArtifact(ctx, "bench.plano", src)
		if err != nil {
			b.Fatal(err)
		}
		if artifact.SchemaVersion != compilerpkg.ArtifactSchemaVersion {
			b.Fatalf("artifact schema version = %q", artifact.SchemaVersion)
		}
	}
}

func BenchmarkCompileStringDetailedWithDiagnostics(b *testing.B) {
	compiler := newRegisteredCompiler(b)
	src := benchmarkDiagnosticSource()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := compiler.CompileStringDetailed(ctx, "errors.plano", src)
		if !result.Diagnostics.HasError() {
			b.Fatal("expected diagnostics")
		}
	}
}

func BenchmarkCompileStringDetailedExprWarmCache(b *testing.B) {
	compiler := newExprBenchmarkCompiler(b, compilerpkg.Options{})
	src := benchmarkExprSource(24)
	ctx := context.Background()

	warm := compiler.CompileStringDetailed(ctx, "expr.plano", src)
	if warm.Diagnostics.HasError() {
		b.Fatalf("unexpected diagnostics: %v", warm.Diagnostics)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := compiler.CompileStringDetailed(ctx, "expr.plano", src)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkCompileStringDetailedExprCacheDisabled(b *testing.B) {
	compiler := newExprBenchmarkCompiler(b, compilerpkg.Options{ExprCacheEntries: -1})
	src := benchmarkExprSource(24)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := compiler.CompileStringDetailed(ctx, "expr.plano", src)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkCompileStringDetailedExprColdCache(b *testing.B) {
	src := benchmarkExprSource(24)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		compiler := newExprBenchmarkCompiler(b, compilerpkg.Options{})
		result := compiler.CompileStringDetailed(ctx, "expr.plano", src)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkCompileFileDetailedWarmCache(b *testing.B) {
	root, imported := benchmarkFiles(b)
	compiler := newRegisteredCompiler(b)
	ctx := context.Background()

	warm := compiler.CompileFileDetailed(ctx, root)
	if warm.Diagnostics.HasError() {
		b.Fatalf("unexpected diagnostics: %v", warm.Diagnostics)
	}
	if imported == "" {
		b.Fatal("expected imported fixture path")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := compiler.CompileFileDetailed(ctx, root)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkCompileFileDetailedGlobImportsColdCache(b *testing.B) {
	root := benchmarkGlobFiles(b, 24)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		compiler := newRegisteredCompiler(b)
		result := compiler.CompileFileDetailed(ctx, root)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkCompileFileDetailedColdCache(b *testing.B) {
	root, imported := benchmarkFiles(b)
	ctx := context.Background()
	if imported == "" {
		b.Fatal("expected imported fixture path")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		compiler := newRegisteredCompiler(b)
		result := compiler.CompileFileDetailed(ctx, root)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkBuilddslLower(b *testing.B) {
	compiler := compilerpkg.New(compilerpkg.Options{})
	if err := builddsl.Register(compiler); err != nil {
		b.Fatal(err)
	}
	result := compiler.CompileStringDetailed(context.Background(), "build.plano", artifactFixtureSource())
	if result.Diagnostics.HasError() {
		b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		project, err := builddsl.Lower(result.HIR)
		if err != nil {
			b.Fatal(err)
		}
		if project == nil || project.Tasks.Len() == 0 {
			b.Fatal("expected lowered project")
		}
	}
}

func newExprBenchmarkCompiler(tb testing.TB, opts compilerpkg.Options) *compilerpkg.Compiler {
	tb.Helper()
	compiler := newRegisteredCompilerWithOptions(tb, opts)
	if err := compiler.RegisterExprVar("branch", "main"); err != nil {
		tb.Fatal(err)
	}
	if err := compiler.RegisterExprFunc("slug", benchmarkSlug, func(string) string { return "" }); err != nil {
		tb.Fatal(err)
	}
	return compiler
}

func benchmarkSlug(params ...any) (any, error) {
	if len(params) != 1 {
		return nil, errors.New("slug expects one argument")
	}
	value, ok := params[0].(string)
	if !ok {
		return nil, errors.New("slug expects string")
	}
	return value, nil
}

func BenchmarkArtifactMarshalBinary(b *testing.B) {
	compiler := newRegisteredCompiler(b)
	ctx := context.Background()
	artifact, err := compiler.CompileStringArtifact(ctx, "bench.plano", artifactFixtureSource())
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		data, marshalErr := artifact.MarshalBinary()
		if marshalErr != nil {
			b.Fatal(marshalErr)
		}
		if len(data) == 0 {
			b.Fatal("expected artifact bytes")
		}
	}
}

func BenchmarkArtifactUnmarshalBinary(b *testing.B) {
	compiler := newRegisteredCompiler(b)
	ctx := context.Background()
	artifact, err := compiler.CompileStringArtifact(ctx, "bench.plano", artifactFixtureSource())
	if err != nil {
		b.Fatal(err)
	}
	data, err := artifact.MarshalBinary()
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		var decoded compilerpkg.Artifact
		if unmarshalErr := decoded.UnmarshalBinary(data); unmarshalErr != nil {
			b.Fatal(unmarshalErr)
		}
		if decoded.SchemaVersion != compilerpkg.ArtifactSchemaVersion {
			b.Fatalf("decoded artifact schema version = %q", decoded.SchemaVersion)
		}
	}
}
