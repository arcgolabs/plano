package lsp

import (
	"go/token"
	"path/filepath"

	"github.com/arcgolabs/plano/compiler"
)

func fileForPath(fset *token.FileSet, path string) *token.File {
	if fset == nil {
		return nil
	}
	clean := filepath.Clean(path)
	var found *token.File
	fset.Iterate(func(file *token.File) bool {
		if filepath.Clean(file.Name()) == clean {
			found = file
			return false
		}
		return true
	})
	return found
}

func findUseAt(binding *compiler.Binding, target token.Pos) (compiler.NameUse, bool) {
	if binding == nil || binding.Uses == nil {
		return compiler.NameUse{}, false
	}
	var best compiler.NameUse
	found := false
	for _, item := range binding.Uses.Values() {
		if containsPos(item.Pos, item.End, target) && (!found || spanWidth(item.Pos, item.End) < spanWidth(best.Pos, best.End)) {
			best = item
			found = true
		}
	}
	return best, found
}

func findExprAt(checks *compiler.CheckInfo, target token.Pos) (compiler.ExprCheck, bool) {
	if checks == nil || checks.Exprs == nil {
		return compiler.ExprCheck{}, false
	}
	var best compiler.ExprCheck
	found := false
	for _, item := range checks.Exprs.Values() {
		if containsPos(item.Pos, item.End, target) && (!found || spanWidth(item.Pos, item.End) < spanWidth(best.Pos, best.End)) {
			best = item
			found = true
		}
	}
	return best, found
}

func findLocalDeclAt(binding *compiler.Binding, target token.Pos) (compiler.LocalBinding, bool) {
	if binding == nil || binding.Locals == nil {
		return compiler.LocalBinding{}, false
	}
	for _, item := range binding.Locals.Values() {
		if containsPos(item.Pos, item.End, target) {
			return item, true
		}
	}
	return compiler.LocalBinding{}, false
}

func findConstDeclAt(binding *compiler.Binding, target token.Pos) (compiler.ConstBinding, bool) {
	if binding == nil || binding.Consts == nil {
		return compiler.ConstBinding{}, false
	}
	for _, item := range binding.Consts.Values() {
		if containsPos(item.Pos, item.End, target) {
			return item, true
		}
	}
	return compiler.ConstBinding{}, false
}

func findFunctionDeclAt(binding *compiler.Binding, target token.Pos) (compiler.FunctionBinding, bool) {
	if binding == nil || binding.Functions == nil {
		return compiler.FunctionBinding{}, false
	}
	for _, item := range binding.Functions.Values() {
		if containsPos(item.Pos, item.End, target) {
			return item, true
		}
	}
	return compiler.FunctionBinding{}, false
}

func findSymbolDeclAt(binding *compiler.Binding, target token.Pos) (compiler.Symbol, bool) {
	if binding == nil || binding.Symbols == nil {
		return compiler.Symbol{}, false
	}
	for _, item := range binding.Symbols.Values() {
		if containsPos(item.Pos, item.End, target) {
			return item, true
		}
	}
	return compiler.Symbol{}, false
}

func containsPos(start, end, target token.Pos) bool {
	if !start.IsValid() || !target.IsValid() {
		return false
	}
	if !end.IsValid() || end < start {
		end = start
	}
	return target >= start && target <= end
}

func spanWidth(start, end token.Pos) int {
	if !end.IsValid() || end < start {
		return 0
	}
	return int(end - start)
}
