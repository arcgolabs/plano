package compiler

import (
	"go/token"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

func (c *checker) checkAllConsts() {
	for _, name := range c.binding.Consts.Keys() {
		c.resolveConstType(name)
	}
}

func (c *checker) checkAllUnits(units []parsedUnit) {
	start, end := unitsSpan(units)
	module := c.newScope(ScopeModule, nil, start, end)
	for _, unit := range units {
		fileScope := c.newScope(ScopeFile, module, unit.File.Pos(), unit.File.End())
		c.checkUnit(unit, fileScope)
	}
}

func (c *checker) checkUnit(unit parsedUnit, scope *checkScope) {
	for _, stmt := range unit.File.Statements {
		c.checkStmt(stmt, scope)
	}
}

func (c *checker) checkStmt(stmt ast.Stmt, scope *checkScope) {
	if c.checkDeclStmt(stmt, scope) {
		return
	}
	if c.checkControlStmt(stmt, scope) {
		return
	}
	if form, ok := stmt.(*ast.FormDecl); ok {
		c.checkForm(form, scope)
	}
}

func (c *checker) checkDeclStmt(stmt ast.Stmt, scope *checkScope) bool {
	switch current := stmt.(type) {
	case *ast.ImportDecl:
		return true
	case *ast.ConstDecl:
		c.resolveConstType(current.Name.Name)
	case *ast.LetDecl:
		c.checkLocalDecl(scope, LocalLet, current.Name, current.Type, current.Value)
	case *ast.FnDecl:
		c.checkFunction(current, scope)
	case *ast.ReturnStmt:
		c.checkExpr(current.Value, scope)
	case *ast.BreakStmt:
		c.checkLoopControl(current.Pos(), current.End(), "break", scope)
	case *ast.ContinueStmt:
		c.checkLoopControl(current.Pos(), current.End(), "continue", scope)
	default:
		return false
	}
	return true
}

func (c *checker) checkControlStmt(stmt ast.Stmt, scope *checkScope) bool {
	switch current := stmt.(type) {
	case *ast.IfStmt:
		c.checkIf(current, scope, nil, nil)
	case *ast.ForStmt:
		c.checkFor(current, scope, nil, nil)
	default:
		return false
	}
	return true
}

func (c *checker) checkFunction(fn *ast.FnDecl, parent *checkScope) {
	scope := c.newScope(ScopeFunction, parent, fn.Pos(), fn.End())
	expected := normalizeType(convertTypeExpr(fn.Result))
	for _, param := range fn.Params {
		c.bindLocal(scope, LocalParam, param.Name, convertTypeExpr(param.Type))
	}
	if fn.Body == nil {
		return
	}
	for _, item := range fn.Body.Items {
		c.checkFunctionItem(item, scope, expected)
	}
}

func (c *checker) checkFunctionItem(item ast.FormItem, scope *checkScope, expectedReturn schema.Type) {
	if c.checkFunctionBindingItem(item, scope) {
		return
	}
	if c.checkFunctionControlItem(item, scope, expectedReturn) {
		return
	}
	switch current := item.(type) {
	case *ast.Assignment:
		c.checkLocalAssignment(current, scope)
	case *ast.CallStmt, *ast.FormDecl:
		c.diagnostics.AddError(current.Pos(), current.End(), "unsupported function body item")
	case *ast.ImportDecl:
		c.diagnostics.AddError(current.Pos(), current.End(), "import is not allowed in function bodies")
	case *ast.FnDecl:
		c.diagnostics.AddError(current.Pos(), current.End(), "nested function declarations are not implemented")
	}
}

func (c *checker) checkForm(form *ast.FormDecl, parent *checkScope) {
	spec, ok := c.compiler.forms.Get(form.Head.String())
	if !ok {
		name := form.Head.String()
		c.diagnostics.AddErrorCodeSuggestions(
			diag.CodeUnknownForm,
			form.Pos(),
			form.End(),
			`unknown form "`+name+`"`,
			c.formSuggestions(name, form.Head.Pos(), form.Head.End())...,
		)
		return
	}
	scope := c.newScope(ScopeForm, parent, form.Pos(), form.End())
	if form.Body == nil {
		return
	}
	for _, item := range form.Body.Items {
		c.checkFormItem(item, scope, spec)
	}
}

func (c *checker) checkFormItem(item ast.FormItem, scope *checkScope, spec schema.FormSpec) {
	if c.checkFormStatementItem(item, scope, spec) {
		return
	}
	if c.checkFormScriptItem(item, scope, spec) {
		return
	}
	switch current := item.(type) {
	case *ast.ReturnStmt:
		c.diagnostics.AddError(current.Pos(), current.End(), "return is not allowed in form bodies")
	case *ast.FnDecl:
		c.diagnostics.AddError(current.Pos(), current.End(), "nested function declarations are not implemented")
	case *ast.ImportDecl:
		c.diagnostics.AddError(current.Pos(), current.End(), "import is not allowed in form bodies")
	}
}

func (c *checker) checkIf(stmt *ast.IfStmt, scope *checkScope, expectedReturn schema.Type, formSpec *schema.FormSpec) {
	condition := c.checkExpr(stmt.Condition, scope)
	if !isTypeAssignable(schema.TypeBool, condition) {
		c.diagnostics.AddErrorCode(diag.CodeTypeMismatch, stmt.Condition.Pos(), stmt.Condition.End(), typeMismatchError("if condition", schema.TypeBool, condition).Error())
	}
	c.checkBlock(stmt.Then, scope, expectedReturn, formSpec)
	c.checkBlock(stmt.Else, scope, expectedReturn, formSpec)
}

func (c *checker) checkFor(stmt *ast.ForStmt, scope *checkScope, expectedReturn schema.Type, formSpec *schema.FormSpec) {
	iterable := c.checkExpr(stmt.Iterable, scope)
	if !isIterableType(iterable) {
		c.diagnostics.AddError(stmt.Iterable.Pos(), stmt.Iterable.End(), "for loop expects list or map")
	}
	loopScope := c.newScope(ScopeLoop, scope, stmt.Pos(), stmt.End())
	if stmt.Index != nil {
		c.bindLocal(loopScope, LocalLoop, stmt.Index, inferIterationKeyType(iterable))
	}
	c.bindLocal(loopScope, LocalLoop, stmt.Name, inferIterationType(iterable))
	if stmt.Filter != nil {
		filter := c.checkExpr(stmt.Filter, loopScope)
		if !isTypeAssignable(schema.TypeBool, filter) {
			c.diagnostics.AddErrorCode(diag.CodeTypeMismatch, stmt.Filter.Pos(), stmt.Filter.End(), typeMismatchError("for where clause", schema.TypeBool, filter).Error())
		}
	}
	c.checkBlock(stmt.Body, loopScope, expectedReturn, formSpec)
}

func (c *checker) checkBlock(block *ast.Block, scope *checkScope, expectedReturn schema.Type, formSpec *schema.FormSpec) {
	if block == nil {
		return
	}
	blockScope := c.newScope(ScopeBlock, scope, block.Pos(), block.End())
	for _, item := range block.Items {
		if formSpec != nil {
			c.checkFormItem(item, blockScope, *formSpec)
			continue
		}
		if expectedReturn != nil {
			c.checkFunctionItem(item, blockScope, expectedReturn)
			continue
		}
		c.checkFormItem(item, blockScope, schema.FormSpec{Name: "block", BodyMode: schema.BodyScript})
	}
}

func (c *checker) newScope(kind ScopeKind, parent *checkScope, pos, end token.Pos) *checkScope {
	parentID := ""
	if parent != nil {
		parentID = parent.id
	}
	return &checkScope{
		id: c.scopeID(checkScopeKey{
			kind:     kind,
			parentID: parentID,
			pos:      pos,
			end:      end,
		}),
		kind:   kind,
		parent: parent,
		locals: mapping.NewMap[string, checkLocalBinding](),
	}
}

func (c *checker) bindLocal(scope *checkScope, kind LocalBindingKind, name *ast.Ident, typ schema.Type) {
	if scope == nil || name == nil {
		return
	}
	scope.locals.Set(name.Name, checkLocalBinding{
		kind: kind,
		typ:  normalizeType(typ),
	})
}

func isIterableType(typ schema.Type) bool {
	switch normalizeType(typ).(type) {
	case schema.ListType, schema.MapType:
		return true
	default:
		return normalizeType(typ) == schema.TypeAny
	}
}

func inferIterationType(typ schema.Type) schema.Type {
	switch current := normalizeType(typ).(type) {
	case schema.ListType:
		return normalizeType(current.Elem)
	case schema.MapType:
		return normalizeType(current.Elem)
	default:
		return schema.TypeAny
	}
}

func inferIterationKeyType(typ schema.Type) schema.Type {
	switch normalizeType(typ).(type) {
	case schema.ListType:
		return schema.TypeInt
	case schema.MapType:
		return schema.TypeString
	default:
		return schema.TypeAny
	}
}
