package compiler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	compilerpkg "github.com/arcgolabs/plano/compiler"
)

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

func benchmarkFiles(tb testing.TB) (string, string) {
	tb.Helper()
	dir := tb.TempDir()
	imported := filepath.Join(dir, "tasks.plano")
	root := filepath.Join(dir, "build.plano")
	writeBenchmarkFile(tb, imported, `
task prepare {}
`)
	writeBenchmarkFile(tb, root, `
import "tasks.plano"

workspace {
  name = "bench"
  default = build
}

task build {
  deps = [prepare]

  run {
    exec("go", "test", "./...")
  }
}
`)
	return root, imported
}

func writeBenchmarkFile(tb testing.TB, path, contents string) {
	tb.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		tb.Fatal(err)
	}
}
