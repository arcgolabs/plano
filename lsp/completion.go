package lsp

import (
	"bytes"
	"go/token"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/prefix"
	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/schema"
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
	offset int
	prefix string
	rng    Range
	line   completionLine
}

type completionScope struct {
	ids      *set.Set[string]
	formKind string
}

type completionLine struct {
	startOnly bool
}

type completionIndex struct {
	items *mapping.OrderedMap[string, CompletionItem]
	trie  *prefix.Trie[CompletionItem]
}

func (s Snapshot) CompletionAt(pos Position) (CompletionList, bool) {
	if ctx, ok := s.exprLangCompletionContext(pos); ok {
		index := newCompletionIndex()
		s.addExprLangCompletions(index)
		return CompletionList{
			Range: ctx.rng,
			Items: index.match(ctx.prefix),
		}, true
	}
	ctx, ok := s.completionContext(pos)
	if !ok {
		return CompletionList{}, false
	}
	scope := s.completionScope(ctx.target)
	index := s.buildCompletionIndex(ctx, scope)
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
		offset: offset,
		prefix: string(src[start:offset]),
		rng: Range{
			Start: positionFromOffset(src, start),
			End:   positionFromOffset(src, end),
		},
		line: completionLine{
			startOnly: lineHasOnlyWhitespace(src, start),
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
	if scopeID == "" {
		return ""
	}
	if s.Result.Binding != nil && s.Result.Binding.Scopes != nil {
		if scope, ok := s.Result.Binding.Scopes.Get(scopeID); ok && scope.FormKind != "" {
			return scope.FormKind
		}
	}
	if s.Result.HIR == nil {
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

func (s Snapshot) buildCompletionIndex(ctx completionContext, scope completionScope) *completionIndex {
	index := newCompletionIndex()
	formKind := scope.formKind
	spec, hasSpec := s.completionFormSpec(formKind)
	if !hasSpec {
		formKind = s.formKindFromSource(ctx)
		spec, hasSpec = s.completionFormSpec(formKind)
	}
	switch {
	case ctx.line.startOnly && hasSpec:
		s.addFormBodyCompletions(index, ctx.target, scope, formKind, spec)
	case ctx.line.startOnly:
		s.addTopLevelCompletions(index)
	default:
		s.addExpressionCompletions(index, ctx.target, scope)
		if hasSpec && spec.BodyMode == schema.BodyScript {
			s.addFieldCompletions(index, formKind, true)
			s.addNestedFormCompletions(index, formKind)
		}
	}
	return index
}

func (s Snapshot) completionFormSpec(formKind string) (schema.FormSpec, bool) {
	if formKind == "" || s.compiler == nil {
		return schema.FormSpec{}, false
	}
	return s.compiler.FormSpec(formKind)
}

func (s Snapshot) addExpressionCompletions(index *completionIndex, target token.Pos, scope completionScope) {
	s.addLocalCompletions(index, target, scope)
	s.addConstCompletions(index)
	s.addFunctionCompletions(index)
	s.addSymbolCompletions(index)
	s.addBuiltinFunctionCompletions(index)
	s.addGlobalCompletions(index)
	s.addExpressionKeywords(index)
}

func (s Snapshot) addTopLevelCompletions(index *completionIndex) {
	s.addFormCompletions(index)
	s.addTopLevelKeywords(index)
}

func (s Snapshot) addFormBodyCompletions(
	index *completionIndex,
	target token.Pos,
	scope completionScope,
	formKind string,
	spec schema.FormSpec,
) {
	switch spec.BodyMode {
	case schema.BodyFieldOnly:
		s.addFieldCompletions(index, formKind, false)
	case schema.BodyFormOnly:
		s.addNestedFormCompletions(index, formKind)
	case schema.BodyMixed:
		s.addFieldCompletions(index, formKind, false)
		s.addNestedFormCompletions(index, formKind)
	case schema.BodyCallOnly:
		s.addActionCompletions(index)
	case schema.BodyScript:
		s.addFieldCompletions(index, formKind, false)
		s.addNestedFormCompletions(index, formKind)
		s.addActionCompletions(index)
		s.addExpressionCompletions(index, target, scope)
		s.addScriptKeywords(index)
	}
}

func (s Snapshot) formKindFromSource(ctx completionContext) string {
	src, ok := s.source(s.Path)
	if !ok {
		return ""
	}
	file, ok := s.fileForPath(s.Path)
	if !ok || file == nil {
		return ""
	}
	return inferFormKindFromSource(file, src, ctx.offset)
}

func lineHasOnlyWhitespace(src []byte, offset int) bool {
	start := offset
	for start > 0 && src[start-1] != '\n' {
		start--
	}
	return len(bytes.TrimSpace(src[start:offset])) == 0
}
