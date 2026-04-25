package compiler

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func inferLiteralExpr(expr ast.Expr) (schema.Type, bool) {
	switch expr.(type) {
	case *ast.StringLiteral:
		return schema.TypeString, true
	case *ast.IntLiteral:
		return schema.TypeInt, true
	case *ast.FloatLiteral:
		return schema.TypeFloat, true
	case *ast.BoolLiteral:
		return schema.TypeBool, true
	case *ast.NullLiteral:
		return schema.TypeAny, true
	case *ast.DurationLiteral:
		return schema.TypeDuration, true
	case *ast.SizeLiteral:
		return schema.TypeSize, true
	default:
		return nil, false
	}
}

func (c *checker) inferCollectionExpr(expr ast.Expr, scope *checkScope) (schema.Type, bool) {
	switch current := expr.(type) {
	case *ast.ArrayExpr:
		return schema.ListType{Elem: c.checkExprList(current.Elements, scope)}, true
	case *ast.ObjectExpr:
		return schema.MapType{Elem: c.checkObjectEntries(current.Entries, scope)}, true
	default:
		return nil, false
	}
}
