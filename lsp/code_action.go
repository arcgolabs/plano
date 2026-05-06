package lsp

import (
	"cmp"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
)

func (s Snapshot) CodeActions(rng Range) list.List[CodeAction] {
	out := list.NewList[CodeAction]()
	diagnostics := s.Diagnostics.Values()
	for index := range diagnostics {
		diagnostic := diagnostics[index]
		if !rangeOverlaps(diagnostic.Range, rng) {
			continue
		}
		actions := s.codeActionsForDiagnostic(diagnostic)
		for _, action := range actions.Values() {
			out.Add(action)
		}
	}
	return *out
}

func (s Snapshot) codeActionsForDiagnostic(item Diagnostic) list.List[CodeAction] {
	out := list.NewListWithCapacity[CodeAction](item.Suggestions.Len())
	for index := range item.Suggestions.Len() {
		suggestion, _ := item.Suggestions.Get(index)
		out.Add(CodeAction{
			Title:       suggestion.Title,
			Kind:        CodeActionQuickFix,
			Diagnostics: *list.NewList(item),
			Edit: WorkspaceEdit{
				Changes: singleTextEdit(s.URI, suggestion.Range, suggestion.Replacement),
			},
			IsPreferred: index == 0,
		})
	}
	return *out
}

func singleTextEdit(
	uri string,
	rng Range,
	newText string,
) *mapping.OrderedMap[string, list.List[TextEdit]] {
	changes := mapping.NewOrderedMap[string, list.List[TextEdit]]()
	changes.Set(uri, *list.NewList(TextEdit{
		Range:   rng,
		NewText: newText,
	}))
	return changes
}

func rangeOverlaps(left, right Range) bool {
	return comparePosition(left.Start, right.End) <= 0 &&
		comparePosition(right.Start, left.End) <= 0
}

func comparePosition(left, right Position) int {
	if order := cmp.Compare(left.Line, right.Line); order != 0 {
		return order
	}
	return cmp.Compare(left.Character, right.Character)
}
