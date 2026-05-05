package lsp

import (
	"cmp"
	"slices"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
)

func (s Snapshot) FoldingRanges() list.List[FoldingRange] {
	return s.cachedFoldingRanges()
}

func (s Snapshot) buildFoldingRanges() list.List[FoldingRange] {
	if s.Result.Binding == nil || s.Result.Binding.Scopes == nil {
		return list.List[FoldingRange]{}
	}
	ranges := make([]FoldingRange, 0, s.Result.Binding.Scopes.Len())
	s.Result.Binding.Scopes.Range(func(_ string, scope compiler.ScopeBinding) bool {
		if !foldableScopeKind(scope.Kind) {
			return true
		}
		rng, ok := s.rangeForSpan(scope.Pos, scope.End)
		if !ok || rng.Start.Line == rng.End.Line {
			return true
		}
		ranges = append(ranges, FoldingRange{
			Range: rng,
			Kind:  FoldingRangeRegion,
		})
		return true
	})
	slices.SortFunc(ranges, compareFoldingRanges)
	return *list.NewList(ranges...)
}

func foldableScopeKind(kind compiler.ScopeKind) bool {
	switch kind {
	case compiler.ScopeForm,
		compiler.ScopeFunction,
		compiler.ScopeBlock,
		compiler.ScopeLoop:
		return true
	case compiler.ScopeModule,
		compiler.ScopeFile:
		return false
	default:
		return false
	}
}

func compareFoldingRanges(left, right FoldingRange) int {
	if order := comparePosition(left.Range.Start, right.Range.Start); order != 0 {
		return order
	}
	if order := comparePosition(left.Range.End, right.Range.End); order != 0 {
		return order
	}
	return cmp.Compare(left.Kind, right.Kind)
}
