package compiler

import "go/token"

type spanKey struct {
	pos token.Pos
	end token.Pos
}

type scopeSpanKey struct {
	kind ScopeKind
	pos  token.Pos
	end  token.Pos
}

func buildScopeSpanIndex(binding *Binding) map[scopeSpanKey]string {
	index := make(map[scopeSpanKey]string, binding.Scopes.Len())
	binding.Scopes.Range(func(_ string, scope ScopeBinding) bool {
		index[scopeSpanKey{
			kind: scope.Kind,
			pos:  scope.Pos,
			end:  scope.End,
		}] = scope.ID
		return true
	})
	return index
}

func buildFieldCheckIndex(checks *CheckInfo) map[spanKey]FieldCheck {
	index := make(map[spanKey]FieldCheck, checks.Fields.Len())
	checks.Fields.Range(func(_ string, field FieldCheck) bool {
		index[spanKey{pos: field.Pos, end: field.End}] = field
		return true
	})
	return index
}

func buildCallCheckIndex(checks *CheckInfo) map[spanKey]CallCheck {
	index := make(map[spanKey]CallCheck, checks.Calls.Len())
	checks.Calls.Range(func(_ string, call CallCheck) bool {
		index[spanKey{pos: call.Pos, end: call.End}] = call
		return true
	})
	return index
}
