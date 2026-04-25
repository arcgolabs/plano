package compiler

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (c *checker) checkFunctionBindingItem(item ast.FormItem, scope *checkScope) bool {
	switch current := item.(type) {
	case *ast.ConstDecl:
		c.checkLocalDecl(scope, current.Name, current.Type, current.Value)
	case *ast.LetDecl:
		c.checkLocalDecl(scope, current.Name, current.Type, current.Value)
	default:
		return false
	}
	return true
}

func (c *checker) checkFunctionControlItem(item ast.FormItem, scope *checkScope, expectedReturn schema.Type) bool {
	switch current := item.(type) {
	case *ast.IfStmt:
		c.checkIf(current, scope, expectedReturn, nil)
	case *ast.ForStmt:
		c.checkFor(current, scope, expectedReturn, nil)
	case *ast.ReturnStmt:
		actual := c.checkExpr(current.Value, scope)
		if expectedReturn != schema.TypeAny && !isTypeAssignable(expectedReturn, actual) {
			c.diagnostics.AddError(current.Pos(), current.End(), typeMismatchError("return", expectedReturn, actual).Error())
		}
	default:
		return false
	}
	return true
}

func (c *checker) checkFormStatementItem(item ast.FormItem, scope *checkScope, spec schema.FormSpec) bool {
	switch current := item.(type) {
	case *ast.Assignment:
		c.checkAssignment(current, scope, spec)
	case *ast.FormDecl:
		if !allowsForm(spec.BodyMode) {
			c.diagnostics.AddError(current.Pos(), current.End(), spec.Name+" does not allow nested forms in "+spec.BodyMode.String()+" body")
			return true
		}
		c.checkForm(current, scope)
	case *ast.CallStmt:
		if !allowsCall(spec.BodyMode) {
			c.diagnostics.AddError(current.Pos(), current.End(), spec.Name+" does not allow call statements in "+spec.BodyMode.String()+" body")
			return true
		}
		c.checkActionCall(current, scope)
	default:
		return false
	}
	return true
}

func (c *checker) checkFormScriptItem(item ast.FormItem, scope *checkScope, spec schema.FormSpec) bool {
	switch current := item.(type) {
	case *ast.ConstDecl:
		c.checkScriptDecl(scope, spec, current.Name, current.Type, current.Value)
	case *ast.LetDecl:
		c.checkScriptDecl(scope, spec, current.Name, current.Type, current.Value)
	case *ast.IfStmt:
		c.checkScriptIf(scope, spec, current)
	case *ast.ForStmt:
		c.checkScriptFor(scope, spec, current)
	default:
		return false
	}
	return true
}
