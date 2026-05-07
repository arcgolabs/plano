package compiler_test

import (
	"context"
	"testing"
)

func BenchmarkCompileStringDetailedControlFlow(b *testing.B) {
	compiler := newRegisteredCompiler(b)
	src := benchmarkControlFlowSource(18)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := compiler.CompileStringDetailed(ctx, "control_flow.plano", src)
		if result.Diagnostics.HasError() {
			b.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
		}
	}
}

func BenchmarkCompileFileDetailedDeepImportGraphWarmCache(b *testing.B) {
	root := benchmarkDeepImportGraphFiles(b, 4, 3)
	compiler := newRegisteredCompiler(b)
	ctx := context.Background()

	warm := compiler.CompileFileDetailed(ctx, root)
	if warm.Diagnostics.HasError() {
		b.Fatalf("unexpected diagnostics: %v", warm.Diagnostics)
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

func BenchmarkCompileFileDetailedDeepImportGraphColdCache(b *testing.B) {
	root := benchmarkDeepImportGraphFiles(b, 4, 3)
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
