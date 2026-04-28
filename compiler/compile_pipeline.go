package compiler

import (
	"go/token"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
)

func (c *Compiler) newCompileState(fset *token.FileSet, index boundIndex, checks *CheckInfo) *compileState {
	return &compileState{
		compiler:    c,
		binding:     index.binding,
		fset:        fset,
		checks:      checks,
		symbols:     index.symbols,
		constDecls:  index.constDecls,
		constValues: mapping.NewOrderedMap[string, any](),
		funcDecls:   index.funcDecls,
		resolving:   mapping.NewOrderedMap[string, bool](),
		scopeIndex:  buildScopeSpanIndex(index.binding),
		fieldIndex:  buildFieldCheckIndex(checks),
		callIndex:   buildCallCheckIndex(checks),
		hir: &HIR{
			Symbols: index.binding.Symbols.Clone(),
			Consts:  mapping.NewOrderedMap[string, HIRConst](),
		},
	}
}

func (s *compileState) resolveAllConsts() {
	for _, name := range s.constDecls.Keys() {
		s.resolveConst(name)
	}
}

func (s *compileState) populateHIRConsts() {
	s.binding.Consts.Range(func(name string, item ConstBinding) bool {
		value, _ := s.constValues.Get(name)
		typ := item.Type
		if typ == nil {
			typ = staticTypeOfValue(value)
		}
		s.hir.Consts.Set(name, HIRConst{
			Name:  name,
			Type:  typ,
			Value: value,
			Pos:   item.Pos,
			End:   item.End,
		})
		return true
	})
}

func (s *compileState) newDocument() *Document {
	return &Document{
		Symbols: s.symbols.Clone(),
		Consts:  s.constValues.Clone(),
	}
}

func (s *compileState) compileTopLevelForms(units []parsedUnit, doc *Document) {
	modulePos, moduleEnd := unitsSpan(units)
	moduleEnv := s.newScopeEnv(nil, ScopeModule, modulePos, moduleEnd)
	for _, unit := range units {
		s.compileUnitForms(unit, moduleEnv, doc)
	}
}

func (s *compileState) compileUnitForms(unit parsedUnit, moduleEnv *env, doc *Document) {
	fileEnv := s.newScopeEnv(moduleEnv, ScopeFile, unit.File.Pos(), unit.File.End())
	for _, stmt := range unit.File.Statements {
		if form, ok := stmt.(*ast.FormDecl); ok {
			s.appendCompiledForm(form, fileEnv, doc)
			continue
		}
		if !isSkippableTopLevel(stmt) {
			s.diags.AddError(stmt.Pos(), stmt.End(), "top-level statement is not supported by the compiler yet")
		}
	}
}

func (s *compileState) appendCompiledForm(form *ast.FormDecl, fileEnv *env, doc *Document) {
	compiled, hirForm := s.compileForm(form, fileEnv)
	if compiled == nil || hirForm == nil {
		return
	}
	doc.Forms.Add(*compiled)
	s.hir.Forms.Add(*hirForm)
}

func isSkippableTopLevel(stmt ast.Stmt) bool {
	switch stmt.(type) {
	case *ast.ImportDecl, *ast.ConstDecl, *ast.FnDecl:
		return true
	default:
		return false
	}
}
