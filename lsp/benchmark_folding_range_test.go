package lsp_test

import "testing"

func BenchmarkSnapshotFoldingRanges(b *testing.B) {
	snapshot, _ := benchmarkLargeSnapshot(b)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		ranges := snapshot.FoldingRanges()
		if ranges.Len() == 0 {
			b.Fatal("expected folding ranges")
		}
	}
}
