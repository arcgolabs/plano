package lsp

import (
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
)

type renameTarget struct {
	name string
	rng  Range
}

func (s Snapshot) PrepareRenameAt(pos Position) (Range, bool) {
	target, ok := s.renameTargetAt(pos)
	if !ok {
		return Range{}, false
	}
	return target.rng, true
}

func (s Snapshot) RenameAt(pos Position, newName string) (WorkspaceEdit, bool) {
	target, ok := s.renameTargetAt(pos)
	if !ok || !isValidRenameIdentifier(newName) {
		return WorkspaceEdit{}, false
	}
	locations, ok := s.ReferencesAt(pos, true)
	if !ok {
		return WorkspaceEdit{}, false
	}
	if newName == target.name {
		return renameWorkspaceEdit(locations, newName), true
	}
	return renameWorkspaceEdit(locations, newName), true
}

func (s Snapshot) renameTargetAt(pos Position) (renameTarget, bool) {
	targetPos, ok := s.renameTokenPos(pos)
	if !ok {
		return renameTarget{}, false
	}
	if target, ok := s.renameUseTarget(targetPos); ok {
		return target, true
	}
	return s.renameDeclarationTarget(targetPos)
}

func (s Snapshot) renameTokenPos(pos Position) (token.Pos, bool) {
	targetPos, ok := s.tokenPos(pos)
	if !ok || s.Result.Binding == nil {
		return token.NoPos, false
	}
	return targetPos, true
}

func (s Snapshot) renameUseTarget(targetPos token.Pos) (renameTarget, bool) {
	use, ok := findUseAt(s.Result.Binding, targetPos)
	if !ok || use.TargetID == "" || !renameableUseKind(use.Kind) {
		return renameTarget{}, false
	}
	rng, ok := s.rangeForSpan(use.Pos, use.End)
	if !ok {
		return renameTarget{}, false
	}
	return renameTarget{name: use.Name, rng: rng}, true
}

func (s Snapshot) renameDeclarationTarget(targetPos token.Pos) (renameTarget, bool) {
	if item, ok := findLocalDeclAt(s.Result.Binding, targetPos); ok {
		return s.renameDeclTarget(compiler.UseLocal, item.Name, item.Pos, item.End)
	}
	if item, ok := findConstDeclAt(s.Result.Binding, targetPos); ok {
		return s.renameDeclTarget(compiler.UseConst, item.Name, item.Pos, item.End)
	}
	if item, ok := findFunctionDeclAt(s.Result.Binding, targetPos); ok {
		return s.renameDeclTarget(compiler.UseFunction, item.Name, item.Pos, item.End)
	}
	if item, ok := findSymbolDeclAt(s.Result.Binding, targetPos); ok {
		return s.renameDeclTarget(compiler.UseSymbol, item.Name, item.Pos, item.End)
	}
	return renameTarget{}, false
}

func (s Snapshot) renameDeclTarget(kind compiler.NameUseKind, name string, pos, end token.Pos) (renameTarget, bool) {
	rng, ok := s.rangeForSpan(pos, end)
	if !ok || !renameableUseKind(kind) {
		return renameTarget{}, false
	}
	return renameTarget{
		name: name,
		rng:  rng,
	}, true
}

func renameableUseKind(kind compiler.NameUseKind) bool {
	switch kind {
	case compiler.UseLocal, compiler.UseConst, compiler.UseFunction, compiler.UseSymbol:
		return true
	case compiler.UseBuiltinFunction, compiler.UseGlobal, compiler.UseAction, compiler.UseUnresolved:
		return false
	default:
		return false
	}
}

func renameWorkspaceEdit(locations list.List[Location], newName string) WorkspaceEdit {
	changes := mapping.NewOrderedMap[string, list.List[TextEdit]]()
	for index := range locations.Len() {
		location, _ := locations.Get(index)
		edits, ok := changes.Get(location.URI)
		if !ok {
			edits = list.List[TextEdit]{}
		}
		editList := list.NewList(edits.Values()...)
		editList.Add(TextEdit{
			Range:   location.Range,
			NewText: newName,
		})
		changes.Set(location.URI, *editList)
	}
	return WorkspaceEdit{Changes: changes}
}

func isValidRenameIdentifier(name string) bool {
	if strings.TrimSpace(name) != name || name == "" {
		return false
	}
	for index, r := range name {
		switch {
		case index == 0 && !isRenameIdentStart(r):
			return false
		case index > 0 && !isRenameIdentContinue(r):
			return false
		}
	}
	return utf8.ValidString(name)
}

func isRenameIdentStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

func isRenameIdentContinue(r rune) bool {
	return isRenameIdentStart(r) || unicode.IsDigit(r)
}
