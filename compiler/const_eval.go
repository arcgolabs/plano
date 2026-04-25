package compiler

import (
	"fmt"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (s *compileState) resolveConst(name string) (any, bool) {
	if value, ok := s.constValues.Get(name); ok {
		return value, true
	}
	decl, ok := s.constDecls.Get(name)
	if !ok {
		return nil, false
	}
	if s.isResolvingConst(name) {
		s.diags.AddError(decl.Pos(), decl.End(), fmt.Sprintf("constant cycle detected at %q", name))
		return nil, false
	}
	s.resolving.Set(name, true)
	value, err := s.evalExpr(decl.Value, nil)
	s.resolving.Delete(name)
	if err != nil {
		s.diags.AddError(decl.Pos(), decl.End(), err.Error())
		return nil, false
	}
	if err := validateConstValue(decl.Type, value); err != nil {
		s.diags.AddError(decl.Pos(), decl.End(), fmt.Sprintf("const %q: %v", name, err))
		return nil, false
	}
	s.constValues.Set(name, value)
	return value, true
}

func (s *compileState) isResolvingConst(name string) bool {
	resolving, ok := s.resolving.Get(name)
	return ok && resolving
}

func validateConstValue(typeExpr ast.TypeExpr, value any) error {
	if typeExpr == nil {
		return nil
	}
	typ := convertTypeExpr(typeExpr)
	if typ == nil {
		return nil
	}
	if err := schema.CheckAssignable(typ, value); err != nil {
		return fmt.Errorf("type check constant: %w", err)
	}
	return nil
}
