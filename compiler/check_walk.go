package compiler

import (
	"go/token"

	"github.com/arcgolabs/plano/ast"
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
	switch current := stmt.(type) {
	case *ast.ImportDecl:
		return
	case *ast.ConstDecl:
		c.resolveConstType(current.Name.Name)
	case *ast.LetDecl:
		c.checkLocalDecl(scope, current.Name, current.Type, current.Value)
	case *ast.FnDecl:
		c.checkFunction(current, scope)
	case *ast.ReturnStmt:
		c.checkExpr(current.Value, scope)
	case *ast.IfStmt:
		c.checkIf(current, scope, nil, nil)
	case *ast.ForStmt:
		c.checkFor(current, scope, nil, nil)
	case *ast.FormDecl:
		c.checkForm(current, scope)
	}
}

func (c *checker) checkFunction(fn *ast.FnDecl, parent *checkScope) {
	scope := c.newScope(ScopeFunction, parent, fn.Pos(), fn.End())
	expected := normalizeType(convertTypeExpr(fn.Result))
	for _, param := range fn.Params {
		c.bindLocal(scope, param.Name, convertTypeExpr(param.Type))
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
	case *ast.Assignment, *ast.CallStmt, *ast.FormDecl:
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
		c.diagnostics.AddError(form.Pos(), form.End(), `unknown form "`+form.Head.String()+`"`)
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

func (c *checker) checkAssignment(assign *ast.Assignment, scope *checkScope, spec schema.FormSpec) {
	if !allowsField(spec.BodyMode) {
		c.diagnostics.AddError(assign.Pos(), assign.End(), spec.Name+" does not allow fields in "+spec.BodyMode.String()+" body")
		return
	}
	fieldSpec, ok := spec.Fields[assign.Name.Name]
	if !ok {
		c.diagnostics.AddError(assign.Pos(), assign.End(), `field "`+assign.Name.Name+`" is not allowed in `+spec.Name)
		return
	}
	actual := c.checkExpr(assign.Value, scope)
	c.recordField(spec.Name, fieldSpec.Name, scope.id, fieldSpec.Type, actual, assign.Pos(), assign.End())
	if !isTypeAssignable(fieldSpec.Type, actual) {
		c.diagnostics.AddError(assign.Pos(), assign.End(), typeMismatchError(`field "`+fieldSpec.Name+`"`, fieldSpec.Type, actual).Error())
	}
}

func (c *checker) checkScriptDecl(scope *checkScope, spec schema.FormSpec, name *ast.Ident, typeExpr ast.TypeExpr, value ast.Expr) {
	if spec.BodyMode != schema.BodyScript {
		c.diagnostics.AddError(value.Pos(), value.End(), spec.Name+" does not allow script statements in "+spec.BodyMode.String()+" body")
		return
	}
	c.checkLocalDecl(scope, name, typeExpr, value)
}

func (c *checker) checkScriptIf(scope *checkScope, spec schema.FormSpec, stmt *ast.IfStmt) {
	if spec.BodyMode != schema.BodyScript {
		c.diagnostics.AddError(stmt.Pos(), stmt.End(), spec.Name+" does not allow script statements in "+spec.BodyMode.String()+" body")
		return
	}
	c.checkIf(stmt, scope, nil, &spec)
}

func (c *checker) checkScriptFor(scope *checkScope, spec schema.FormSpec, stmt *ast.ForStmt) {
	if spec.BodyMode != schema.BodyScript {
		c.diagnostics.AddError(stmt.Pos(), stmt.End(), spec.Name+" does not allow script statements in "+spec.BodyMode.String()+" body")
		return
	}
	c.checkFor(stmt, scope, nil, &spec)
}

func (c *checker) checkLocalDecl(scope *checkScope, name *ast.Ident, typeExpr ast.TypeExpr, value ast.Expr) {
	actual := c.checkExpr(value, scope)
	declared := convertTypeExpr(typeExpr)
	if declared != nil && !isTypeAssignable(declared, actual) {
		c.diagnostics.AddError(value.Pos(), value.End(), typeMismatchError(`binding "`+name.Name+`"`, declared, actual).Error())
	}
	if declared == nil {
		declared = actual
	}
	c.bindLocal(scope, name, declared)
}

func (c *checker) checkIf(stmt *ast.IfStmt, scope *checkScope, expectedReturn schema.Type, formSpec *schema.FormSpec) {
	condition := c.checkExpr(stmt.Condition, scope)
	if !isTypeAssignable(schema.TypeBool, condition) {
		c.diagnostics.AddError(stmt.Condition.Pos(), stmt.Condition.End(), typeMismatchError("if condition", schema.TypeBool, condition).Error())
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
	c.bindLocal(loopScope, stmt.Name, inferIterationType(iterable))
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
		id: c.scopeIndex[checkScopeKey{
			kind:     kind,
			parentID: parentID,
			pos:      pos,
			end:      end,
		}],
		parent: parent,
		locals: make(map[string]schema.Type),
	}
}

func (c *checker) bindLocal(scope *checkScope, name *ast.Ident, typ schema.Type) {
	if scope == nil || name == nil {
		return
	}
	scope.locals[name.Name] = normalizeType(typ)
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
