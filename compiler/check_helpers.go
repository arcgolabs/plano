package compiler

import (
	"go/token"
	"strconv"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/lo"
)

func (c *checker) resolveConstTypeInScope(name string, scope *checkScope) schema.Type {
	if typ, ok := c.constTypes.Get(name); ok {
		return typ
	}
	decl, ok := c.constDecls.Get(name)
	if !ok {
		return schema.TypeAny
	}
	if c.resolving.Contains(name) {
		c.diagnostics.AddError(decl.Pos(), decl.End(), `constant cycle detected at "`+name+`"`)
		return schema.TypeAny
	}
	c.resolving.Add(name)
	if scope == nil {
		scope = c.newScope(ScopeFile, nil, decl.Pos(), decl.End())
	}
	actual := c.checkExpr(decl.Value, scope)
	declared := convertTypeExpr(decl.Type)
	if declared != nil && !isTypeAssignable(declared, actual) {
		c.diagnostics.AddErrorCode(diag.CodeTypeMismatch, decl.Pos(), decl.End(), typeMismatchError(`const "`+name+`"`, declared, actual).Error())
	}
	if declared == nil {
		declared = actual
	}
	resolved := normalizeType(declared)
	c.constTypes.Set(name, resolved)
	c.resolving.Remove(name)
	return resolved
}

func signatureArgType(index int, paramTypes list.List[schema.Type], variadicType schema.Type) schema.Type {
	if index < paramTypes.Len() {
		if typ, ok := paramTypes.Get(index); ok {
			return typ
		}
	}
	return variadicType
}

func paramTypes(params list.List[ParamBinding]) list.List[schema.Type] {
	items := lo.Map(params.Values(), func(param ParamBinding, _ int) schema.Type {
		return param.Type
	})
	return *list.NewList(items...)
}

func (c *checker) recordExpr(expr ast.Expr, kind string, scope *checkScope, typ schema.Type) schema.Type {
	c.nextExpr++
	id := "expr-" + strconv.Itoa(c.nextExpr)
	scopeID := ""
	if scope != nil {
		scopeID = scope.id
	}
	c.checks.Exprs.Set(id, ExprCheck{
		ID:      id,
		Kind:    kind,
		ScopeID: scopeID,
		Type:    normalizeType(typ),
		Pos:     expr.Pos(),
		End:     expr.End(),
	})
	return normalizeType(typ)
}

func (c *checker) recordField(formKind, field, scopeID string, expected, actual schema.Type, pos, end token.Pos) {
	c.nextField++
	id := "field-" + strconv.Itoa(c.nextField)
	c.checks.Fields.Set(id, FieldCheck{
		ID:       id,
		FormKind: formKind,
		Field:    field,
		ScopeID:  scopeID,
		Expected: normalizeType(expected),
		Actual:   normalizeType(actual),
		Pos:      pos,
		End:      end,
	})
}

func (c *checker) recordCall(name, scopeID string, args []schema.Type, result schema.Type, pos, end token.Pos) schema.Type {
	c.nextCall++
	id := "call-" + strconv.Itoa(c.nextCall)
	c.checks.Calls.Set(id, CallCheck{
		ID:      id,
		Name:    name,
		ScopeID: scopeID,
		Args:    *list.NewList(args...),
		Result:  normalizeType(result),
		Pos:     pos,
		End:     end,
	})
	return normalizeType(result)
}

func exprKind(expr ast.Expr) string {
	if kind, ok := literalExprKind(expr); ok {
		return kind
	}
	if kind, ok := structuralExprKind(expr); ok {
		return kind
	}
	if kind, ok := accessExprKind(expr); ok {
		return kind
	}
	return "expr"
}

func literalExprKind(expr ast.Expr) (string, bool) {
	switch expr.(type) {
	case *ast.StringLiteral:
		return "string", true
	case *ast.IntLiteral:
		return "int", true
	case *ast.FloatLiteral:
		return "float", true
	case *ast.BoolLiteral:
		return "bool", true
	case *ast.NullLiteral:
		return "null", true
	case *ast.DurationLiteral:
		return "duration", true
	case *ast.SizeLiteral:
		return "size", true
	default:
		return "", false
	}
}

func structuralExprKind(expr ast.Expr) (string, bool) {
	switch expr.(type) {
	case *ast.IdentExpr:
		return "ident", true
	case *ast.ArrayExpr:
		return "array", true
	case *ast.ObjectExpr:
		return "object", true
	case *ast.ParenExpr:
		return "paren", true
	case *ast.UnaryExpr:
		return "unary", true
	case *ast.BinaryExpr:
		return "binary", true
	default:
		return "", false
	}
}

func accessExprKind(expr ast.Expr) (string, bool) {
	switch expr.(type) {
	case *ast.SelectorExpr:
		return "selector", true
	case *ast.IndexExpr:
		return "index", true
	case *ast.CallExpr:
		return "call", true
	default:
		return "", false
	}
}

func findCheckLocal(scope *checkScope, name string) (checkLocalBinding, bool) {
	for current := scope; current != nil; current = current.parent {
		if binding, ok := current.locals.Get(name); ok {
			return binding, true
		}
	}
	return checkLocalBinding{}, false
}
