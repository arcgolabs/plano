package compiler

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

func (c *checker) checkAssignment(assign *ast.Assignment, scope *checkScope, spec schema.FormSpec) {
	if spec.BodyMode == schema.BodyScript && c.shouldAssignLocal(spec, assign.Name.Name, scope) {
		c.checkLocalAssignment(assign, scope)
		return
	}
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

func (c *checker) checkScriptDecl(scope *checkScope, spec schema.FormSpec, kind LocalBindingKind, name *ast.Ident, typeExpr ast.TypeExpr, value ast.Expr) {
	if spec.BodyMode != schema.BodyScript {
		c.diagnostics.AddError(value.Pos(), value.End(), spec.Name+" does not allow script statements in "+spec.BodyMode.String()+" body")
		return
	}
	if _, ok := spec.Fields[name.Name]; ok {
		c.diagnostics.AddError(value.Pos(), value.End(), `binding "`+name.Name+`" conflicts with field "`+name.Name+`" in `+spec.Name)
		return
	}
	c.checkLocalDecl(scope, kind, name, typeExpr, value)
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
	if _, ok := spec.Fields[stmt.Name.Name]; ok {
		c.diagnostics.AddError(stmt.Pos(), stmt.End(), `loop variable "`+stmt.Name.Name+`" conflicts with field "`+stmt.Name.Name+`" in `+spec.Name)
		return
	}
	c.checkFor(stmt, scope, nil, &spec)
}

func (c *checker) checkLocalDecl(scope *checkScope, kind LocalBindingKind, name *ast.Ident, typeExpr ast.TypeExpr, value ast.Expr) {
	actual := c.checkExpr(value, scope)
	declared := convertTypeExpr(typeExpr)
	if declared != nil && !isTypeAssignable(declared, actual) {
		c.diagnostics.AddError(value.Pos(), value.End(), typeMismatchError(`binding "`+name.Name+`"`, declared, actual).Error())
	}
	if declared == nil {
		declared = actual
	}
	c.bindLocal(scope, kind, name, declared)
}

func (c *checker) checkLocalAssignment(assign *ast.Assignment, scope *checkScope) {
	actual := c.checkExpr(assign.Value, scope)
	binding, ok := findCheckLocal(scope, assign.Name.Name)
	if !ok {
		c.diagnostics.AddError(assign.Pos(), assign.End(), `undefined local binding "`+assign.Name.Name+`"`)
		return
	}
	if binding.kind == LocalConst {
		c.diagnostics.AddError(assign.Pos(), assign.End(), `cannot assign to const "`+assign.Name.Name+`"`)
		return
	}
	if !isTypeAssignable(binding.typ, actual) {
		c.diagnostics.AddError(assign.Pos(), assign.End(), typeMismatchError(`assignment "`+assign.Name.Name+`"`, binding.typ, actual).Error())
	}
}

func (c *checker) shouldAssignLocal(spec schema.FormSpec, name string, scope *checkScope) bool {
	if _, ok := spec.Fields[name]; ok {
		return false
	}
	_, ok := findCheckLocal(scope, name)
	return ok
}
