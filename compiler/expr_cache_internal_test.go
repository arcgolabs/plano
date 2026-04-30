package compiler

import (
	"testing"

	"github.com/expr-lang/expr/vm"
)

func TestExprCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache := newExprCache(1)
	first := &vm.Program{}
	second := &vm.Program{}

	cache.add("first", first)
	cache.add("second", second)

	if _, ok := cache.get("first"); ok {
		t.Fatal("expected first program to be evicted")
	}
	got, ok := cache.get("second")
	if !ok {
		t.Fatal("expected second program to remain cached")
	}
	if got != second {
		t.Fatal("cached program mismatch")
	}
}

func TestNormalizeExprCacheEntries(t *testing.T) {
	if got := normalizeExprCacheEntries(-1); got != 0 {
		t.Fatalf("disabled entries = %d, want 0", got)
	}
	if got := normalizeExprCacheEntries(0); got != defaultExprCacheEntries {
		t.Fatalf("default entries = %d, want %d", got, defaultExprCacheEntries)
	}
	if got := normalizeExprCacheEntries(8); got != 8 {
		t.Fatalf("custom entries = %d, want 8", got)
	}
}
