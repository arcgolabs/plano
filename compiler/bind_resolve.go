package compiler

import (
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
)

type scopeFrame struct {
	id     string
	parent *scopeFrame
	locals *mapping.Map[string, string]
}

func (b *binder) resolveUnits(units []parsedUnit) {
	start, end := unitsSpan(units)
	module := b.newScope(ScopeModule, nil, start, end)
	for _, unit := range units {
		fileScope := b.newScope(ScopeFile, module, unit.File.Pos(), unit.File.End())
		b.resolveUnit(unit, fileScope)
	}
}

func (b *binder) resolveUnit(unit parsedUnit, scope *scopeFrame) {
	for _, stmt := range unit.File.Statements {
		b.resolveStmt(stmt, scope)
	}
}

func (b *binder) resolveStmt(stmt ast.Stmt, scope *scopeFrame) {
	switch current := stmt.(type) {
	case *ast.ImportDecl:
		return
	case *ast.ConstDecl:
		b.resolveExpr(current.Value, scope)
	case *ast.LetDecl:
		b.resolveLocalDecl(scope, LocalLet, current.Name, current.Type, current.Value)
	case *ast.FnDecl:
		b.resolveFunction(current, scope)
	case *ast.ReturnStmt:
		b.resolveExpr(current.Value, scope)
	case *ast.BreakStmt, *ast.ContinueStmt:
		return
	case *ast.IfStmt:
		b.resolveIf(current, scope)
	case *ast.ForStmt:
		b.resolveFor(current, scope)
	case *ast.FormDecl:
		b.resolveForm(current, scope)
	}
}

func (b *binder) resolveFunction(fn *ast.FnDecl, parent *scopeFrame) {
	functionScope := b.newScope(ScopeFunction, parent, fn.Pos(), fn.End())
	for _, param := range fn.Params {
		b.bindLocal(functionScope, LocalParam, param.Name, param.Type)
	}
	if fn.Body == nil {
		return
	}
	for _, item := range fn.Body.Items {
		b.resolveFormItem(item, functionScope)
	}
}

func (b *binder) resolveForm(form *ast.FormDecl, parent *scopeFrame) {
	formScope := b.newScope(ScopeForm, parent, form.Pos(), form.End())
	if form.Body == nil {
		return
	}
	for _, item := range form.Body.Items {
		b.resolveFormItem(item, formScope)
	}
}

func (b *binder) resolveFormItem(item ast.FormItem, scope *scopeFrame) {
	if b.resolveBindingItem(item, scope) {
		return
	}
	if b.resolveControlItem(item, scope) {
		return
	}
	switch current := item.(type) {
	case *ast.Assignment:
		b.recordAssignableUse(current.Name, scope)
		b.resolveExpr(current.Value, scope)
	case *ast.FormDecl:
		b.resolveForm(current, scope)
	case *ast.CallStmt:
		b.recordActionUse(current.Callee, scope)
		for _, arg := range current.Args {
			b.resolveExpr(arg, scope)
		}
	case *ast.ImportDecl:
		return
	}
}

func (b *binder) resolveLocalDecl(scope *scopeFrame, kind LocalBindingKind, name *ast.Ident, typeExpr ast.TypeExpr, value ast.Expr) {
	b.resolveExpr(value, scope)
	b.bindLocal(scope, kind, name, typeExpr)
}

func (b *binder) resolveIf(stmt *ast.IfStmt, scope *scopeFrame) {
	b.resolveExpr(stmt.Condition, scope)
	b.resolveBlock(stmt.Then, scope)
	b.resolveBlock(stmt.Else, scope)
}

func (b *binder) resolveFor(stmt *ast.ForStmt, scope *scopeFrame) {
	b.resolveExpr(stmt.Iterable, scope)
	loopScope := b.newScope(ScopeLoop, scope, stmt.Pos(), stmt.End())
	if stmt.Index != nil {
		b.bindLocal(loopScope, LocalLoop, stmt.Index, nil)
	}
	b.bindLocal(loopScope, LocalLoop, stmt.Name, nil)
	b.resolveBlock(stmt.Body, loopScope)
}

func (b *binder) resolveBlock(block *ast.Block, scope *scopeFrame) {
	if block == nil {
		return
	}
	blockScope := b.newScope(ScopeBlock, scope, block.Pos(), block.End())
	for _, item := range block.Items {
		b.resolveFormItem(item, blockScope)
	}
}

func (b *binder) resolveExpr(expr ast.Expr, scope *scopeFrame) {
	if b.resolveLiteralExpr(expr) {
		return
	}
	if b.resolveCompositeExpr(expr, scope) {
		return
	}
	b.resolveAccessExpr(expr, scope)
}

func (b *binder) resolveBindingItem(item ast.FormItem, scope *scopeFrame) bool {
	switch current := item.(type) {
	case *ast.ConstDecl:
		b.resolveLocalDecl(scope, LocalConst, current.Name, current.Type, current.Value)
	case *ast.LetDecl:
		b.resolveLocalDecl(scope, LocalLet, current.Name, current.Type, current.Value)
	case *ast.FnDecl:
		b.resolveFunction(current, scope)
	case *ast.ReturnStmt:
		b.resolveExpr(current.Value, scope)
	case *ast.BreakStmt, *ast.ContinueStmt:
		return true
	default:
		return false
	}
	return true
}

func (b *binder) resolveControlItem(item ast.FormItem, scope *scopeFrame) bool {
	switch current := item.(type) {
	case *ast.IfStmt:
		b.resolveIf(current, scope)
	case *ast.ForStmt:
		b.resolveFor(current, scope)
	case *ast.BreakStmt, *ast.ContinueStmt:
		return true
	default:
		return false
	}
	return true
}

func (b *binder) resolveLiteralExpr(expr ast.Expr) bool {
	switch expr.(type) {
	case nil, *ast.StringLiteral, *ast.IntLiteral, *ast.FloatLiteral, *ast.BoolLiteral, *ast.NullLiteral, *ast.DurationLiteral, *ast.SizeLiteral:
		return true
	default:
		return false
	}
}

func (b *binder) resolveCompositeExpr(expr ast.Expr, scope *scopeFrame) bool {
	switch current := expr.(type) {
	case *ast.IdentExpr:
		b.recordIdentUse(current.Name, scope)
	case *ast.ArrayExpr:
		for _, item := range current.Elements {
			b.resolveExpr(item, scope)
		}
	case *ast.ObjectExpr:
		for _, entry := range current.Entries {
			b.resolveExpr(entry.Value, scope)
		}
	case *ast.ParenExpr:
		b.resolveExpr(current.X, scope)
	case *ast.UnaryExpr:
		b.resolveExpr(current.X, scope)
	case *ast.BinaryExpr:
		b.resolveExpr(current.X, scope)
		b.resolveExpr(current.Y, scope)
	default:
		return false
	}
	return true
}

func (b *binder) resolveAccessExpr(expr ast.Expr, scope *scopeFrame) {
	switch current := expr.(type) {
	case *ast.SelectorExpr:
		b.resolveExpr(current.X, scope)
	case *ast.IndexExpr:
		b.resolveExpr(current.X, scope)
		b.resolveExpr(current.Index, scope)
	case *ast.CallExpr:
		if name, ok := callName(current.Fun); ok {
			b.recordCallableUse(name, current.Fun.Pos(), current.Fun.End(), scope)
		} else {
			b.resolveExpr(current.Fun, scope)
		}
		for _, arg := range current.Args {
			b.resolveExpr(arg, scope)
		}
	}
}
