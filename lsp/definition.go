package lsp

import (
	"go/token"

	"github.com/arcgolabs/plano/compiler"
)

func (s Snapshot) DefinitionAt(pos Position) (Location, bool) {
	target, ok := s.tokenPos(pos)
	if !ok || s.Result.Binding == nil {
		return Location{}, false
	}
	if use, ok := findUseAt(s.Result.Binding, target); ok {
		return s.definitionLocation(use)
	}
	return s.declarationLocation(target)
}

func (s Snapshot) declarationLocation(target token.Pos) (Location, bool) {
	if item, ok := findLocalDeclAt(s.Result.Binding, target); ok {
		return s.locationForSpan(item.Pos, item.End)
	}
	if item, ok := findConstDeclAt(s.Result.Binding, target); ok {
		return s.locationForSpan(item.Pos, item.End)
	}
	if item, ok := findFunctionDeclAt(s.Result.Binding, target); ok {
		return s.locationForSpan(item.Pos, item.End)
	}
	if item, ok := findSymbolDeclAt(s.Result.Binding, target); ok {
		return s.locationForSpan(item.Pos, item.End)
	}
	return Location{}, false
}

func (s Snapshot) definitionLocation(use compiler.NameUse) (Location, bool) {
	switch use.Kind {
	case compiler.UseLocal:
		return s.localLocation(use.TargetID)
	case compiler.UseConst:
		return s.constLocation(use.TargetID)
	case compiler.UseFunction:
		return s.functionLocation(use.TargetID)
	case compiler.UseSymbol:
		return s.symbolLocation(use.TargetID)
	case compiler.UseBuiltinFunction, compiler.UseGlobal, compiler.UseAction, compiler.UseUnresolved:
		return Location{}, false
	default:
		return Location{}, false
	}
}

func (s Snapshot) localLocation(id string) (Location, bool) {
	item, ok := s.Result.Binding.Locals.Get(id)
	if !ok {
		return Location{}, false
	}
	return s.locationForSpan(item.Pos, item.End)
}

func (s Snapshot) constLocation(id string) (Location, bool) {
	item, ok := s.Result.Binding.Consts.Get(id)
	if !ok {
		return Location{}, false
	}
	return s.locationForSpan(item.Pos, item.End)
}

func (s Snapshot) functionLocation(id string) (Location, bool) {
	item, ok := s.Result.Binding.Functions.Get(id)
	if !ok {
		return Location{}, false
	}
	return s.locationForSpan(item.Pos, item.End)
}

func (s Snapshot) symbolLocation(id string) (Location, bool) {
	item, ok := s.Result.Binding.Symbols.Get(id)
	if !ok {
		return Location{}, false
	}
	return s.locationForSpan(item.Pos, item.End)
}
