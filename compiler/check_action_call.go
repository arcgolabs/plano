package compiler

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

func (c *checker) checkActionCall(call *ast.CallStmt, scope *checkScope) {
	argTypes := make([]schema.Type, 0, len(call.Args))
	for _, arg := range call.Args {
		argTypes = append(argTypes, c.checkExpr(arg, scope))
	}

	spec, ok := c.compiler.actions.Get(call.Callee.String())
	if !ok {
		name := call.Callee.String()
		c.diagnostics.AddErrorCodeSuggestions(
			diag.CodeUnknownAction,
			call.Pos(),
			call.End(),
			`unknown action "`+name+`"`,
			c.actionSuggestions(name, call.Callee.Pos(), call.Callee.End())...,
		)
		c.recordCall(call.Callee.String(), scope.id, argTypes, schema.TypeAny, call.Pos(), call.End())
		return
	}

	c.checkSignature("action", call.Callee.String(), spec.MinArgs, spec.MaxArgs, spec.ArgTypes, spec.VariadicType, argTypes, call.Pos(), call.End())
	c.recordCall(call.Callee.String(), scope.id, argTypes, schema.TypeAny, call.Pos(), call.End())
}
