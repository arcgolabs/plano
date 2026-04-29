package lsp

import (
	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
)

func (s Snapshot) ReferencesAt(pos Position, includeDeclaration bool) (list.List[Location], bool) {
	target, ok := s.referenceTargetAt(pos)
	if !ok {
		return list.List[Location]{}, false
	}
	return s.cachedReferences(target, includeDeclaration)
}

type referenceTarget struct {
	kind compiler.NameUseKind
	id   string
}

func (s Snapshot) referenceTargetAt(pos Position) (referenceTarget, bool) {
	target, ok := s.tokenPos(pos)
	if !ok || s.Result.Binding == nil {
		return referenceTarget{}, false
	}
	if use, ok := findUseAt(s.Result.Binding, target); ok && use.TargetID != "" {
		return referenceTarget{
			kind: use.Kind,
			id:   use.TargetID,
		}, true
	}
	if item, ok := findLocalDeclAt(s.Result.Binding, target); ok {
		return referenceTarget{kind: compiler.UseLocal, id: item.ID}, true
	}
	if item, ok := findConstDeclAt(s.Result.Binding, target); ok {
		return referenceTarget{kind: compiler.UseConst, id: item.Name}, true
	}
	if item, ok := findFunctionDeclAt(s.Result.Binding, target); ok {
		return referenceTarget{kind: compiler.UseFunction, id: item.Name}, true
	}
	if item, ok := findSymbolDeclAt(s.Result.Binding, target); ok {
		return referenceTarget{kind: compiler.UseSymbol, id: item.Name}, true
	}
	return referenceTarget{}, false
}

func (s Snapshot) referenceDeclarationLocation(target referenceTarget) (Location, bool) {
	switch target.kind {
	case compiler.UseLocal:
		return s.localLocation(target.id)
	case compiler.UseConst:
		return s.constLocation(target.id)
	case compiler.UseFunction:
		return s.functionLocation(target.id)
	case compiler.UseSymbol:
		return s.symbolLocation(target.id)
	case compiler.UseBuiltinFunction, compiler.UseGlobal, compiler.UseAction, compiler.UseUnresolved:
		return Location{}, false
	default:
		return Location{}, false
	}
}
