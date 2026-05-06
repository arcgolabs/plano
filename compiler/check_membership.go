package compiler

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

func (c *checker) checkMembershipOp(expr *ast.BinaryExpr, left, right schema.Type) schema.Type {
	switch current := normalizeType(right).(type) {
	case schema.ListType:
		c.checkListMembership(expr, left, current.Elem)
	case schema.MapType:
		c.checkMapMembership(expr, left)
	case schema.BuiltinType:
		if current != schema.TypeAny {
			c.diagnostics.AddError(expr.Pos(), expr.End(), "operator in expects list or map")
		}
	default:
		c.diagnostics.AddError(expr.Pos(), expr.End(), "operator in expects list or map")
	}
	return schema.TypeBool
}

func (c *checker) checkListMembership(expr *ast.BinaryExpr, item, elem schema.Type) {
	if !isTypeAssignable(elem, item) {
		c.diagnostics.AddErrorCode(
			diag.CodeTypeMismatch,
			expr.X.Pos(),
			expr.X.End(),
			typeMismatchError("operator in item", elem, item).Error(),
		)
	}
}

func (c *checker) checkMapMembership(expr *ast.BinaryExpr, key schema.Type) {
	if !isTypeAssignable(schema.TypeString, key) {
		c.diagnostics.AddErrorCode(
			diag.CodeTypeMismatch,
			expr.X.Pos(),
			expr.X.End(),
			typeMismatchError("operator in map key", schema.TypeString, key).Error(),
		)
	}
}
