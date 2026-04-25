package compiler

import (
	"go/token"
	"strconv"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (c *checker) resolveConstType(name string) schema.Type {
	return c.resolveConstTypeInScope(name, c.newScope(ScopeFile, nil, token.NoPos, token.NoPos))
}

func (c *checker) checkExpr(expr ast.Expr, scope *checkScope) schema.Type {
	if expr == nil {
		return schema.TypeAny
	}
	typ := c.inferExpr(expr, scope)
	return c.recordExpr(expr, exprKind(expr), scope, typ)
}

func (c *checker) inferExpr(expr ast.Expr, scope *checkScope) schema.Type {
	if typ, ok := inferLiteralExpr(expr); ok {
		return typ
	}
	if typ, ok := c.inferCollectionExpr(expr, scope); ok {
		return typ
	}
	return c.inferComputedExpr(expr, scope)
}

func (c *checker) inferIdent(name *ast.Ident, scope *checkScope) schema.Type {
	if name == nil {
		return schema.TypeAny
	}
	if local, ok := findCheckLocal(scope, name.Name); ok {
		return normalizeType(local)
	}
	if value, ok := c.compiler.globals.Get(name.Name); ok {
		return staticTypeOfValue(value)
	}
	if _, ok := c.binding.Consts.Get(name.Name); ok {
		return c.resolveConstType(name.Name)
	}
	if symbol, ok := c.binding.Symbols.Get(name.Name); ok {
		return schema.RefType{Kind: symbol.Kind}
	}
	c.diagnostics.AddError(name.Pos(), name.End(), `undefined symbol "`+name.Name+`"`)
	return schema.TypeAny
}

func (c *checker) checkExprList(items []ast.Expr, scope *checkScope) schema.Type {
	types := make([]schema.Type, 0, len(items))
	for _, item := range items {
		types = append(types, c.checkExpr(item, scope))
	}
	return mergeTypes(types)
}

func (c *checker) checkObjectEntries(entries []*ast.ObjectEntry, scope *checkScope) schema.Type {
	types := make([]schema.Type, 0, len(entries))
	for _, entry := range entries {
		types = append(types, c.checkExpr(entry.Value, scope))
	}
	return mergeTypes(types)
}

func (c *checker) checkUnaryExpr(expr *ast.UnaryExpr, scope *checkScope) schema.Type {
	operand := c.checkExpr(expr.X, scope)
	switch expr.Op {
	case "!":
		if !isTypeAssignable(schema.TypeBool, operand) {
			c.diagnostics.AddError(expr.Pos(), expr.End(), typeMismatchError("operator !", schema.TypeBool, operand).Error())
		}
		return schema.TypeBool
	case "-":
		if !isNumericType(operand) {
			c.diagnostics.AddError(expr.Pos(), expr.End(), "operator - expects numeric operand")
			return schema.TypeAny
		}
		return operand
	default:
		return schema.TypeAny
	}
}

func (c *checker) checkBinaryExpr(expr *ast.BinaryExpr, scope *checkScope) schema.Type {
	left := c.checkExpr(expr.X, scope)
	right := c.checkExpr(expr.Y, scope)
	if isArithmeticOp(expr.Op) {
		return c.checkArithmeticOp(expr, left, right)
	}
	if expr.Op == "%" {
		if !isTypeAssignable(schema.TypeInt, left) || !isTypeAssignable(schema.TypeInt, right) {
			c.diagnostics.AddError(expr.Pos(), expr.End(), "operator % expects int operands")
		}
		return schema.TypeInt
	}
	if isEqualityOp(expr.Op) {
		return schema.TypeBool
	}
	if isLogicalOp(expr.Op) {
		return c.checkLogicalOp(expr, left, right)
	}
	if isComparisonOp(expr.Op) {
		return c.checkComparisonOp(expr, left, right)
	}
	return schema.TypeAny
}

func (c *checker) checkArithmeticOp(expr *ast.BinaryExpr, left, right schema.Type) schema.Type {
	if expr.Op == "+" && isStringType(left) && isStringType(right) {
		return schema.TypeString
	}
	if isNumericType(left) && isNumericType(right) {
		if left == schema.TypeFloat || right == schema.TypeFloat {
			return schema.TypeFloat
		}
		return schema.TypeInt
	}
	c.diagnostics.AddError(expr.Pos(), expr.End(), "arithmetic operators expect string/string or number/number")
	return schema.TypeAny
}

func (c *checker) inferComputedExpr(expr ast.Expr, scope *checkScope) schema.Type {
	switch current := expr.(type) {
	case *ast.IdentExpr:
		return c.inferIdent(current.Name, scope)
	case *ast.ParenExpr:
		return c.checkExpr(current.X, scope)
	case *ast.UnaryExpr:
		return c.checkUnaryExpr(current, scope)
	case *ast.BinaryExpr:
		return c.checkBinaryExpr(current, scope)
	case *ast.SelectorExpr:
		return c.checkSelectorExpr(current, scope)
	case *ast.IndexExpr:
		return c.checkIndexExpr(current, scope)
	case *ast.CallExpr:
		return c.checkCallExpr(current, scope)
	default:
		return schema.TypeAny
	}
}

func (c *checker) checkLogicalOp(expr *ast.BinaryExpr, left, right schema.Type) schema.Type {
	if !isTypeAssignable(schema.TypeBool, left) || !isTypeAssignable(schema.TypeBool, right) {
		c.diagnostics.AddError(expr.Pos(), expr.End(), "logical operators expect bool operands")
	}
	return schema.TypeBool
}

func (c *checker) checkComparisonOp(expr *ast.BinaryExpr, left, right schema.Type) schema.Type {
	if (!isNumericType(left) || !isNumericType(right)) && (!isStringType(left) || !isStringType(right)) {
		c.diagnostics.AddError(expr.Pos(), expr.End(), "comparison expects compatible operands")
	}
	return schema.TypeBool
}

func (c *checker) checkSelectorExpr(expr *ast.SelectorExpr, scope *checkScope) schema.Type {
	base := c.checkExpr(expr.X, scope)
	switch current := normalizeType(base).(type) {
	case schema.MapType:
		return normalizeType(current.Elem)
	case schema.BuiltinType:
		if current == schema.TypeAny {
			return schema.TypeAny
		}
	}
	c.diagnostics.AddError(expr.Pos(), expr.End(), "selector requires object value")
	return schema.TypeAny
}

func (c *checker) checkIndexExpr(expr *ast.IndexExpr, scope *checkScope) schema.Type {
	base := c.checkExpr(expr.X, scope)
	index := c.checkExpr(expr.Index, scope)

	switch current := normalizeType(base).(type) {
	case schema.ListType:
		if !isTypeAssignable(schema.TypeInt, index) {
			c.diagnostics.AddError(expr.Index.Pos(), expr.Index.End(), typeMismatchError("array index", schema.TypeInt, index).Error())
		}
		return normalizeType(current.Elem)
	case schema.MapType:
		if !isTypeAssignable(schema.TypeString, index) {
			c.diagnostics.AddError(expr.Index.Pos(), expr.Index.End(), typeMismatchError("object index", schema.TypeString, index).Error())
		}
		return normalizeType(current.Elem)
	default:
		return schema.TypeAny
	}
}

func (c *checker) checkCallExpr(expr *ast.CallExpr, scope *checkScope) schema.Type {
	argTypes := make([]schema.Type, 0, len(expr.Args))
	for _, arg := range expr.Args {
		argTypes = append(argTypes, c.checkExpr(arg, scope))
	}

	name, ok := callName(expr.Fun)
	if !ok {
		c.checkExpr(expr.Fun, scope)
		return c.recordCall("call", scope.id, argTypes, schema.TypeAny, expr.Pos(), expr.End())
	}

	var result schema.Type = schema.TypeAny
	switch {
	case c.checkUserFunctionCall(name, argTypes, expr):
		if fn, ok := c.binding.Functions.Get(name); ok {
			result = normalizeType(fn.Result)
		}
	case c.checkBuiltinFunctionCall(name, argTypes, expr):
		if spec, ok := c.compiler.funcs.Get(name); ok {
			result = normalizeType(spec.Result)
		}
	default:
		c.diagnostics.AddError(expr.Pos(), expr.End(), `unknown function "`+name+`"`)
	}

	c.recordCall(name, scope.id, argTypes, result, expr.Pos(), expr.End())
	return result
}

func (c *checker) checkUserFunctionCall(name string, argTypes []schema.Type, expr *ast.CallExpr) bool {
	fn, ok := c.binding.Functions.Get(name)
	if !ok {
		return false
	}
	c.checkSignature("function", name, len(fn.Params), len(fn.Params), paramTypes(fn.Params), nil, argTypes, expr.Pos(), expr.End())
	return true
}

func (c *checker) checkBuiltinFunctionCall(name string, argTypes []schema.Type, expr *ast.CallExpr) bool {
	spec, ok := c.compiler.funcs.Get(name)
	if !ok {
		return false
	}
	c.checkSignature("function", name, spec.MinArgs, spec.MaxArgs, spec.ParamTypes, spec.VariadicType, argTypes, expr.Pos(), expr.End())
	return true
}

func (c *checker) checkActionCall(call *ast.CallStmt, scope *checkScope) {
	argTypes := make([]schema.Type, 0, len(call.Args))
	for _, arg := range call.Args {
		argTypes = append(argTypes, c.checkExpr(arg, scope))
	}
	spec, ok := c.compiler.actions.Get(call.Callee.String())
	if !ok {
		c.diagnostics.AddError(call.Pos(), call.End(), `unknown action "`+call.Callee.String()+`"`)
		c.recordCall(call.Callee.String(), scope.id, argTypes, schema.TypeAny, call.Pos(), call.End())
		return
	}
	c.checkSignature("action", call.Callee.String(), spec.MinArgs, spec.MaxArgs, spec.ArgTypes, spec.VariadicType, argTypes, call.Pos(), call.End())
	c.recordCall(call.Callee.String(), scope.id, argTypes, schema.TypeAny, call.Pos(), call.End())
}

func (c *checker) checkSignature(kind, name string, minArgs, maxArgs int, paramTypes []schema.Type, variadicType schema.Type, argTypes []schema.Type, pos, end token.Pos) {
	if err := validateArity(kind, name, minArgs, maxArgs, len(argTypes)); err != nil {
		c.diagnostics.AddError(pos, end, err.Error())
		return
	}
	for idx, argType := range argTypes {
		want := signatureArgType(idx, paramTypes, variadicType)
		if want == nil {
			continue
		}
		if !isTypeAssignable(want, argType) {
			c.diagnostics.AddError(pos, end, typeMismatchError(kind+" argument "+strconv.Itoa(idx+1), want, argType).Error())
		}
	}
}
