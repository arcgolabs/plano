package lsp

import (
	"go/token"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/prefix"
	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/compiler"
)

var keywordCompletions = []CompletionItem{
	{Label: "import", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "const", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "let", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "fn", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "return", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "break", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "continue", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "if", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "else", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "for", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "in", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "true", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "false", Kind: CompletionKeyword, Detail: "keyword"},
	{Label: "null", Kind: CompletionKeyword, Detail: "keyword"},
}

type completionContext struct {
	target token.Pos
	prefix string
	rng    Range
}

type completionScope struct {
	ids      *set.Set[string]
	formKind string
}

type completionIndex struct {
	items *mapping.OrderedMap[string, CompletionItem]
	trie  *prefix.Trie[CompletionItem]
}

func (s Snapshot) CompletionAt(pos Position) (CompletionList, bool) {
	ctx, ok := s.completionContext(pos)
	if !ok {
		return CompletionList{}, false
	}
	scope := s.completionScope(ctx.target)
	index := s.buildCompletionIndex(ctx.target, scope)
	return CompletionList{
		Range: ctx.rng,
		Items: index.match(ctx.prefix),
	}, true
}

func (s Snapshot) completionContext(pos Position) (completionContext, bool) {
	src, ok := s.source(s.Path)
	if !ok {
		return completionContext{}, false
	}
	offset, ok := offsetFromPosition(src, pos)
	if !ok {
		return completionContext{}, false
	}
	target, ok := s.tokenPos(pos)
	if !ok {
		return completionContext{}, false
	}
	start, end := completionBounds(src, offset)
	return completionContext{
		target: target,
		prefix: string(src[start:offset]),
		rng: Range{
			Start: positionFromOffset(src, start),
			End:   positionFromOffset(src, end),
		},
	}, true
}

func (s Snapshot) completionScope(target token.Pos) completionScope {
	scope := completionScope{ids: set.NewSet[string]()}
	current, ok := findScopeAt(s.Result.Binding, target)
	if !ok || s.Result.Binding == nil || s.Result.Binding.Scopes == nil {
		return scope
	}
	formScope := ""
	for {
		scope.ids.Add(current.ID)
		if formScope == "" && current.Kind == compiler.ScopeForm {
			formScope = current.ID
		}
		if current.ParentID == "" {
			break
		}
		parent, ok := s.Result.Binding.Scopes.Get(current.ParentID)
		if !ok {
			break
		}
		current = parent
	}
	scope.formKind = s.formKindForScope(formScope)
	return scope
}

func (s Snapshot) formKindForScope(scopeID string) string {
	if scopeID == "" || s.Result.HIR == nil {
		return ""
	}
	return formKindForScope(scopeID, s.Result.HIR.Forms)
}

func formKindForScope(scopeID string, forms list.List[compiler.HIRForm]) string {
	for index := range forms.Len() {
		form, _ := forms.Get(index)
		if form.ScopeID == scopeID {
			return form.Kind
		}
		if nested := formKindForScope(scopeID, form.Forms); nested != "" {
			return nested
		}
	}
	return ""
}

func (s Snapshot) buildCompletionIndex(target token.Pos, scope completionScope) *completionIndex {
	index := newCompletionIndex()
	s.addLocalCompletions(index, target, scope)
	s.addConstCompletions(index)
	s.addFunctionCompletions(index)
	s.addSymbolCompletions(index)
	s.addFieldCompletions(index, scope.formKind)
	s.addFormCompletions(index)
	s.addBuiltinFunctionCompletions(index)
	s.addActionCompletions(index)
	s.addGlobalCompletions(index)
	s.addKeywordCompletions(index)
	return index
}
