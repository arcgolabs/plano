package compiler

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

func (c *checker) checkCollectionBuiltin(name string, argTypes []schema.Type, expr *ast.CallExpr) {
	switch name {
	case "get":
		c.checkGetCall(argTypes, expr)
	case "slice":
		c.checkSliceCall(argTypes, expr)
	case "has":
		c.checkHasCall(argTypes, expr)
	}
}

func (c *checker) checkGetCall(argTypes []schema.Type, expr *ast.CallExpr) {
	if len(argTypes) < 2 {
		return
	}
	base := normalizeType(argTypes[0])
	key := normalizeType(argTypes[1])
	switch current := base.(type) {
	case schema.ListType:
		_ = current
		if !isTypeAssignable(schema.TypeInt, key) {
			c.diagnostics.AddErrorCode(diag.CodeTypeMismatch, expr.Args[1].Pos(), expr.Args[1].End(), typeMismatchError("get index", schema.TypeInt, key).Error())
		}
	case schema.MapType:
		if !isTypeAssignable(schema.TypeString, key) {
			c.diagnostics.AddErrorCode(diag.CodeTypeMismatch, expr.Args[1].Pos(), expr.Args[1].End(), typeMismatchError("get key", schema.TypeString, key).Error())
		}
	default:
		if base != schema.TypeAny {
			c.diagnostics.AddError(expr.Pos(), expr.End(), "get expects list or map")
		}
	}
}

func (c *checker) checkSliceCall(argTypes []schema.Type, expr *ast.CallExpr) {
	if len(argTypes) < 2 {
		return
	}
	base := normalizeType(argTypes[0])
	if _, ok := base.(schema.ListType); !ok && base != schema.TypeAny {
		c.diagnostics.AddError(expr.Args[0].Pos(), expr.Args[0].End(), "slice expects list argument")
	}
	for idx := 1; idx < len(argTypes); idx++ {
		if !isTypeAssignable(schema.TypeInt, argTypes[idx]) {
			c.diagnostics.AddErrorCode(diag.CodeTypeMismatch, expr.Args[idx].Pos(), expr.Args[idx].End(), typeMismatchError("slice index", schema.TypeInt, argTypes[idx]).Error())
		}
	}
}

func (c *checker) checkHasCall(argTypes []schema.Type, expr *ast.CallExpr) {
	if len(argTypes) < 2 {
		return
	}
	base := normalizeType(argTypes[0])
	key := normalizeType(argTypes[1])
	switch base.(type) {
	case schema.ListType:
		return
	case schema.MapType:
		if !isTypeAssignable(schema.TypeString, key) {
			c.diagnostics.AddErrorCode(diag.CodeTypeMismatch, expr.Args[1].Pos(), expr.Args[1].End(), typeMismatchError("has key", schema.TypeString, key).Error())
		}
	default:
		if base != schema.TypeAny {
			c.diagnostics.AddError(expr.Pos(), expr.End(), "has expects list or map")
		}
	}
}
