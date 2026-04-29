package lsp

import (
	"strconv"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/compiler"
)

func (s Snapshot) ReferencesAt(pos Position, includeDeclaration bool) (list.List[Location], bool) {
	target, ok := s.referenceTargetAt(pos)
	if !ok {
		return list.List[Location]{}, false
	}

	seen := set.NewSet[string]()
	locations := list.NewList[Location]()
	s.addReferenceDeclaration(locations, seen, target, includeDeclaration)
	s.addReferenceUses(locations, seen, target)
	return *locations, locations.Len() > 0
}

type referenceTarget struct {
	kind compiler.NameUseKind
	id   string
}

func (s Snapshot) addReferenceDeclaration(
	locations *list.List[Location],
	seen *set.Set[string],
	target referenceTarget,
	includeDeclaration bool,
) {
	if !includeDeclaration {
		return
	}
	location, ok := s.referenceDeclarationLocation(target)
	if ok {
		addReferenceLocation(locations, seen, location)
	}
}

func (s Snapshot) addReferenceUses(
	locations *list.List[Location],
	seen *set.Set[string],
	target referenceTarget,
) {
	if s.Result.Binding == nil || s.Result.Binding.Uses == nil {
		return
	}
	s.Result.Binding.Uses.Range(func(_ string, use compiler.NameUse) bool {
		location, ok := s.referenceUseLocation(use, target)
		if ok {
			addReferenceLocation(locations, seen, location)
		}
		return true
	})
}

func (s Snapshot) referenceUseLocation(use compiler.NameUse, target referenceTarget) (Location, bool) {
	if use.Kind != target.kind || use.TargetID != target.id {
		return Location{}, false
	}
	return s.locationForSpan(use.Pos, use.End)
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

func addReferenceLocation(items *list.List[Location], seen *set.Set[string], location Location) {
	key := referenceLocationKey(location)
	if seen.Contains(key) {
		return
	}
	seen.Add(key)
	items.Add(location)
}

func referenceLocationKey(location Location) string {
	return location.URI +
		":" + positionKey(location.Range.Start) +
		":" + positionKey(location.Range.End)
}

func positionKey(pos Position) string {
	return strconv.Itoa(pos.Line) + ":" + strconv.Itoa(pos.Character)
}
