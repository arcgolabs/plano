package compiler

import (
	"go/token"

	"github.com/arcgolabs/collectionx/mapping"
)

type spanKey struct {
	pos token.Pos
	end token.Pos
}

type scopeSpanKey struct {
	kind ScopeKind
	pos  token.Pos
	end  token.Pos
}

func buildScopeSpanIndex(binding *Binding) *mapping.Map[scopeSpanKey, string] {
	index := mapping.NewMapWithCapacity[scopeSpanKey, string](binding.Scopes.Len())
	binding.Scopes.Range(func(_ string, scope ScopeBinding) bool {
		index.Set(scopeSpanKey{
			kind: scope.Kind,
			pos:  scope.Pos,
			end:  scope.End,
		}, scope.ID)
		return true
	})
	return index
}

func buildFieldCheckIndex(checks *CheckInfo) *mapping.Map[spanKey, FieldCheck] {
	index := mapping.NewMapWithCapacity[spanKey, FieldCheck](checks.Fields.Len())
	checks.Fields.Range(func(_ string, field FieldCheck) bool {
		index.Set(spanKey{pos: field.Pos, end: field.End}, field)
		return true
	})
	return index
}

func buildCallCheckIndex(checks *CheckInfo) *mapping.Map[spanKey, CallCheck] {
	index := mapping.NewMapWithCapacity[spanKey, CallCheck](checks.Calls.Len())
	checks.Calls.Range(func(_ string, call CallCheck) bool {
		index.Set(spanKey{pos: call.Pos, end: call.End}, call)
		return true
	})
	return index
}
