package compiler

import (
	"go/token"
	"strconv"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
)

func (b *binder) bindLocal(scope *scopeFrame, kind LocalBindingKind, name *ast.Ident, typeExpr ast.TypeExpr) {
	if name == nil {
		return
	}
	id := b.nextLocalID()
	scope.locals.Set(name.Name, id)
	b.binding.Locals.Set(id, LocalBinding{
		ID:      id,
		Name:    name.Name,
		Kind:    kind,
		ScopeID: scope.id,
		Type:    convertTypeExpr(typeExpr),
		Pos:     name.Pos(),
		End:     name.End(),
	})
}

func (b *binder) recordIdentUse(name *ast.Ident, scope *scopeFrame) {
	if name == nil {
		return
	}
	useKind, target := b.resolveName(name.Name, scope)
	b.recordUse(name.Name, useKind, scope, target, name.Pos(), name.End())
}

func (b *binder) recordAssignableUse(name *ast.Ident, scope *scopeFrame) {
	if name == nil {
		return
	}
	useKind, target := b.resolveName(name.Name, scope)
	if useKind == UseUnresolved {
		return
	}
	b.recordUse(name.Name, useKind, scope, target, name.Pos(), name.End())
}

func (b *binder) recordCallableUse(name string, pos, end token.Pos, scope *scopeFrame) {
	useKind, target := b.resolveCallable(name)
	b.recordUse(name, useKind, scope, target, pos, end)
}

func (b *binder) recordActionUse(name *ast.QualifiedIdent, scope *scopeFrame) {
	if name == nil {
		return
	}
	kind, target := b.resolveAction(name.String())
	b.recordUse(name.String(), kind, scope, target, name.Pos(), name.End())
}

func (b *binder) resolveName(name string, scope *scopeFrame) (NameUseKind, string) {
	if localID, ok := findLocal(scope, name); ok {
		return UseLocal, localID
	}
	if _, ok := b.compiler.globals.Get(name); ok {
		return UseGlobal, name
	}
	if _, ok := b.binding.Consts.Get(name); ok {
		return UseConst, name
	}
	if _, ok := b.binding.Symbols.Get(name); ok {
		return UseSymbol, name
	}
	return UseUnresolved, ""
}

func (b *binder) resolveCallable(name string) (NameUseKind, string) {
	if _, ok := b.binding.Functions.Get(name); ok {
		return UseFunction, name
	}
	if _, ok := b.compiler.funcs.Get(name); ok {
		return UseBuiltinFunction, name
	}
	return UseUnresolved, ""
}

func (b *binder) resolveAction(name string) (NameUseKind, string) {
	if _, ok := b.compiler.actions.Get(name); ok {
		return UseAction, name
	}
	return UseUnresolved, ""
}

func (b *binder) recordUse(name string, kind NameUseKind, scope *scopeFrame, target string, pos, end token.Pos) {
	id := b.nextUseID()
	scopeID := ""
	if scope != nil {
		scopeID = scope.id
	}
	b.binding.Uses.Set(id, NameUse{
		ID:       id,
		Name:     name,
		Kind:     kind,
		ScopeID:  scopeID,
		TargetID: target,
		Pos:      pos,
		End:      end,
	})
}

func (b *binder) newScope(kind ScopeKind, formKind string, parent *scopeFrame, pos, end token.Pos) *scopeFrame {
	id := b.nextScopeID()
	parentID := ""
	if parent != nil {
		parentID = parent.id
	}
	b.binding.Scopes.Set(id, ScopeBinding{
		ID:       id,
		Kind:     kind,
		FormKind: formKind,
		ParentID: parentID,
		Pos:      pos,
		End:      end,
	})
	return &scopeFrame{
		id:     id,
		parent: parent,
		locals: mapping.NewMap[string, string](),
	}
}

func findLocal(scope *scopeFrame, name string) (string, bool) {
	for current := scope; current != nil; current = current.parent {
		if localID, ok := current.locals.Get(name); ok {
			return localID, true
		}
	}
	return "", false
}

func unitsSpan(units []parsedUnit) (token.Pos, token.Pos) {
	if len(units) == 0 {
		return token.NoPos, token.NoPos
	}
	return units[0].File.Pos(), units[len(units)-1].File.End()
}

func (b *binder) nextScopeID() string {
	b.nextScope++
	return "scope-" + strconv.Itoa(b.nextScope)
}

func (b *binder) nextLocalID() string {
	b.nextLocal++
	return "local-" + strconv.Itoa(b.nextLocal)
}

func (b *binder) nextUseID() string {
	b.nextUse++
	return "use-" + strconv.Itoa(b.nextUse)
}
