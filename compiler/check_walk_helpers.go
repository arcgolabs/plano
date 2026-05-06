package compiler

import (
	"go/token"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

func (c *checker) checkLoopControl(pos, end token.Pos, keyword string, scope *checkScope) {
	if isInsideLoop(scope) {
		return
	}
	c.diagnostics.AddError(pos, end, keyword+" is only allowed inside loops")
}

func (c *checker) checkScriptLoopControl(scope *checkScope, spec schema.FormSpec, pos, end token.Pos, keyword string) {
	if spec.BodyMode != schema.BodyScript {
		c.diagnostics.AddError(pos, end, spec.Name+" does not allow script statements in "+spec.BodyMode.String()+" body")
		return
	}
	c.checkLoopControl(pos, end, keyword, scope)
}

func isInsideLoop(scope *checkScope) bool {
	for current := scope; current != nil; current = current.parent {
		if current.kind == ScopeLoop {
			return true
		}
	}
	return false
}

func (c *checker) checkFunctionBindingItem(item ast.FormItem, scope *checkScope) bool {
	switch current := item.(type) {
	case *ast.ConstDecl:
		c.checkLocalDecl(scope, LocalConst, current.Name, current.Type, current.Value)
	case *ast.LetDecl:
		c.checkLocalDecl(scope, LocalLet, current.Name, current.Type, current.Value)
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
			c.diagnostics.AddErrorCode(diag.CodeTypeMismatch, current.Pos(), current.End(), typeMismatchError("return", expectedReturn, actual).Error())
		}
	case *ast.BreakStmt:
		c.checkLoopControl(current.Pos(), current.End(), "break", scope)
	case *ast.ContinueStmt:
		c.checkLoopControl(current.Pos(), current.End(), "continue", scope)
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
		if !allowsNestedFormName(spec, current.Head.String()) {
			name := current.Head.String()
			c.diagnostics.AddErrorCodeSuggestions(
				diag.CodeUnknownNestedForm,
				current.Pos(),
				current.End(),
				`nested form "`+name+`" is not allowed in `+spec.Name,
				c.nestedFormSuggestions(spec, name, current.Head.Pos(), current.Head.End())...,
			)
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
		c.checkScriptDecl(scope, spec, LocalConst, current.Name, current.Type, current.Value)
	case *ast.LetDecl:
		c.checkScriptDecl(scope, spec, LocalLet, current.Name, current.Type, current.Value)
	case *ast.IfStmt:
		c.checkScriptIf(scope, spec, current)
	case *ast.ForStmt:
		c.checkScriptFor(scope, spec, current)
	case *ast.BreakStmt:
		c.checkScriptLoopControl(scope, spec, current.Pos(), current.End(), "break")
	case *ast.ContinueStmt:
		c.checkScriptLoopControl(scope, spec, current.Pos(), current.End(), "continue")
	default:
		return false
	}
	return true
}
