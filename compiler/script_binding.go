package compiler

import (
	"fmt"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (s *compileState) bindLocalValue(locals *env, kind LocalBindingKind, name string, typeExpr ast.TypeExpr, value any) error {
	typ := normalizeType(convertTypeExpr(typeExpr))
	if typeExpr != nil {
		if err := schema.CheckAssignable(typ, value); err != nil {
			return fmt.Errorf("binding %q: %w", name, err)
		}
	} else {
		typ = staticTypeOfValue(value)
	}
	locals.BindLocal(name, kind, typ, value)
	return nil
}
