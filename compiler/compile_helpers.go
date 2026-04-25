package compiler

import "go/token"

func (s *compileState) newScopeEnv(parent *env, kind ScopeKind, pos, end token.Pos) *env {
	return newEnv(parent, s.scopeID(kind, pos, end))
}

func (s *compileState) scopeID(kind ScopeKind, pos, end token.Pos) string {
	return s.scopeIndex[scopeSpanKey{
		kind: kind,
		pos:  pos,
		end:  end,
	}]
}

func (s *compileState) fieldCheck(pos, end token.Pos) (FieldCheck, bool) {
	check, ok := s.fieldIndex[spanKey{pos: pos, end: end}]
	return check, ok
}

func (s *compileState) callCheck(pos, end token.Pos) (CallCheck, bool) {
	check, ok := s.callIndex[spanKey{pos: pos, end: end}]
	return check, ok
}
