package lsp

import (
	"go/token"
	"slices"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/plano/compiler"
)

func (s Snapshot) DocumentSymbols() list.List[DocumentSymbol] {
	if s.Result.Binding == nil {
		return list.List[DocumentSymbol]{}
	}
	return documentSymbolsFromEntries(s.topLevelDocumentSymbolEntries())
}

type documentSymbolEntry struct {
	pos    token.Pos
	symbol DocumentSymbol
}

func (s Snapshot) topLevelDocumentSymbolEntries() []documentSymbolEntry {
	entries := make([]documentSymbolEntry, 0)
	entries = s.appendConstDocumentSymbols(entries)
	entries = s.appendFunctionDocumentSymbols(entries)
	entries = s.appendFormDocumentSymbols(entries)
	return entries
}

func (s Snapshot) appendConstDocumentSymbols(entries []documentSymbolEntry) []documentSymbolEntry {
	if s.Result.Binding == nil || s.Result.Binding.Consts == nil {
		return entries
	}
	s.Result.Binding.Consts.Range(func(_ string, item compiler.ConstBinding) bool {
		symbol, ok := s.constDocumentSymbol(item)
		if ok {
			entries = append(entries, documentSymbolEntry{pos: item.Pos, symbol: symbol})
		}
		return true
	})
	return entries
}

func (s Snapshot) appendFunctionDocumentSymbols(entries []documentSymbolEntry) []documentSymbolEntry {
	if s.Result.Binding == nil || s.Result.Binding.Functions == nil {
		return entries
	}
	s.Result.Binding.Functions.Range(func(_ string, item compiler.FunctionBinding) bool {
		symbol, ok := s.functionDocumentSymbol(item)
		if ok {
			entries = append(entries, documentSymbolEntry{pos: item.Pos, symbol: symbol})
		}
		return true
	})
	return entries
}

func (s Snapshot) appendFormDocumentSymbols(entries []documentSymbolEntry) []documentSymbolEntry {
	if s.Result.HIR == nil {
		return entries
	}
	for index := range s.Result.HIR.Forms.Len() {
		form, _ := s.Result.HIR.Forms.Get(index)
		symbol, ok := s.formDocumentSymbol(form)
		if ok {
			entries = append(entries, documentSymbolEntry{pos: form.Pos, symbol: symbol})
		}
	}
	return entries
}

func (s Snapshot) constDocumentSymbol(item compiler.ConstBinding) (DocumentSymbol, bool) {
	if !s.isCurrentFileSpan(item.Pos, item.End) {
		return DocumentSymbol{}, false
	}
	rng, ok := s.rangeForSpan(item.Pos, item.End)
	if !ok {
		return DocumentSymbol{}, false
	}
	return DocumentSymbol{
		Name:           item.Name,
		Detail:         typeString(item.Type),
		Kind:           SymbolConst,
		Range:          rng,
		SelectionRange: rng,
	}, true
}

func (s Snapshot) functionDocumentSymbol(item compiler.FunctionBinding) (DocumentSymbol, bool) {
	if !s.isCurrentFileSpan(item.Pos, item.End) {
		return DocumentSymbol{}, false
	}
	rng, ok := s.rangeForSpan(item.Pos, item.End)
	if !ok {
		return DocumentSymbol{}, false
	}
	return DocumentSymbol{
		Name:           item.Name,
		Detail:         "fn",
		Kind:           SymbolFunction,
		Range:          rng,
		SelectionRange: rng,
	}, true
}

func (s Snapshot) formDocumentSymbol(form compiler.HIRForm) (DocumentSymbol, bool) {
	if !s.isCurrentFileSpan(form.Pos, form.End) {
		return DocumentSymbol{}, false
	}
	rng, ok := s.rangeForSpan(form.Pos, form.End)
	if !ok {
		return DocumentSymbol{}, false
	}
	selectionRange := rng
	name := form.Kind
	if form.Symbol != nil {
		name = form.Symbol.Name
		if symbolRange, ok := s.rangeForSpan(form.Symbol.Pos, form.Symbol.End); ok {
			selectionRange = symbolRange
		}
	}

	children := s.formDocumentChildren(form)
	return DocumentSymbol{
		Name:           name,
		Detail:         form.Kind,
		Kind:           SymbolForm,
		Range:          rng,
		SelectionRange: selectionRange,
		Children:       children,
	}, true
}

func (s Snapshot) formDocumentChildren(form compiler.HIRForm) list.List[DocumentSymbol] {
	entries := make([]documentSymbolEntry, 0)
	entries = s.appendFieldDocumentSymbols(entries, form)
	entries = s.appendNestedFormDocumentSymbols(entries, form)
	return documentSymbolsFromEntries(entries)
}

func (s Snapshot) appendFieldDocumentSymbols(entries []documentSymbolEntry, form compiler.HIRForm) []documentSymbolEntry {
	if form.Fields == nil {
		return entries
	}
	form.Fields.Range(func(_ string, field compiler.HIRField) bool {
		symbol, ok := s.fieldDocumentSymbol(form, field)
		if ok {
			entries = append(entries, documentSymbolEntry{pos: field.Pos, symbol: symbol})
		}
		return true
	})
	return entries
}

func (s Snapshot) appendNestedFormDocumentSymbols(entries []documentSymbolEntry, form compiler.HIRForm) []documentSymbolEntry {
	for index := range form.Forms.Len() {
		nested, _ := form.Forms.Get(index)
		symbol, ok := s.formDocumentSymbol(nested)
		if ok {
			entries = append(entries, documentSymbolEntry{pos: nested.Pos, symbol: symbol})
		}
	}
	return entries
}

func (s Snapshot) fieldDocumentSymbol(form compiler.HIRForm, field compiler.HIRField) (DocumentSymbol, bool) {
	if field.Pos == form.Pos && field.End == form.End {
		return DocumentSymbol{}, false
	}
	if !s.isCurrentFileSpan(field.Pos, field.End) {
		return DocumentSymbol{}, false
	}
	rng, ok := s.rangeForSpan(field.Pos, field.End)
	if !ok {
		return DocumentSymbol{}, false
	}
	return DocumentSymbol{
		Name:           field.Name,
		Detail:         typeString(field.Actual),
		Kind:           SymbolField,
		Range:          rng,
		SelectionRange: rng,
	}, true
}

func documentSymbolsFromEntries(entries []documentSymbolEntry) list.List[DocumentSymbol] {
	sortDocumentSymbolEntries(entries)
	items := list.NewListWithCapacity[DocumentSymbol](len(entries))
	for index := range entries {
		items.Add(entries[index].symbol)
	}
	return *items
}

func sortDocumentSymbolEntries(entries []documentSymbolEntry) {
	slices.SortFunc(entries, func(left, right documentSymbolEntry) int {
		if left.pos < right.pos {
			return -1
		}
		if left.pos > right.pos {
			return 1
		}
		return 0
	})
}

func (s Snapshot) isCurrentFileSpan(pos, end token.Pos) bool {
	path, _, ok := s.spanLocation(pos, end)
	return ok && path == s.Path
}
