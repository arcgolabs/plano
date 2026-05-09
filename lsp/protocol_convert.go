package lsp

import (
	"math"

	"github.com/arcgolabs/collectionx/list"
	"go.lsp.dev/protocol"
)

func fromProtocolPosition(pos protocol.Position) Position {
	return Position{
		Line:      int(pos.Line),
		Character: int(pos.Character),
	}
}

func fromProtocolRange(rng protocol.Range) Range {
	return Range{
		Start: fromProtocolPosition(rng.Start),
		End:   fromProtocolPosition(rng.End),
	}
}

func toProtocolPosition(pos Position) protocol.Position {
	return protocol.Position{
		Line:      clampUint32(pos.Line),
		Character: clampUint32(pos.Character),
	}
}

func toProtocolRange(rng Range) protocol.Range {
	return protocol.Range{
		Start: toProtocolPosition(rng.Start),
		End:   toProtocolPosition(rng.End),
	}
}

func toProtocolLocation(location Location) protocol.Location {
	return protocol.Location{
		URI:   protocol.DocumentURI(location.URI),
		Range: toProtocolRange(location.Range),
	}
}

func toProtocolLocations(items list.List[Location]) []protocol.Location {
	return mapList(items, func(item Location) protocol.Location {
		return toProtocolLocation(item)
	})
}

func toProtocolHover(hover Hover) *protocol.Hover {
	rng := toProtocolRange(hover.Range)
	return &protocol.Hover{
		Range: &rng,
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: hover.Contents,
		},
	}
}

func toProtocolWorkspaceEdit(edit WorkspaceEdit) *protocol.WorkspaceEdit {
	changes := make(map[protocol.DocumentURI][]protocol.TextEdit)
	if edit.Changes != nil {
		edit.Changes.Range(func(uri string, items list.List[TextEdit]) bool {
			changes[protocol.DocumentURI(uri)] = toProtocolTextEdits(items)
			return true
		})
	}
	return &protocol.WorkspaceEdit{Changes: changes}
}

func toProtocolCodeActions(items list.List[CodeAction]) []protocol.CodeAction {
	return mapList(items, func(item CodeAction) protocol.CodeAction {
		return protocol.CodeAction{
			Title:       item.Title,
			Kind:        protocol.CodeActionKind(item.Kind),
			Diagnostics: toProtocolDiagnostics(item.Diagnostics),
			Edit:        toProtocolWorkspaceEdit(item.Edit),
			IsPreferred: item.IsPreferred,
		}
	})
}

func toProtocolFoldingRanges(items list.List[FoldingRange]) []protocol.FoldingRange {
	return mapList(items, func(item FoldingRange) protocol.FoldingRange {
		return protocol.FoldingRange{
			StartLine:      clampUint32(item.Range.Start.Line),
			StartCharacter: clampUint32(item.Range.Start.Character),
			EndLine:        clampUint32(item.Range.End.Line),
			EndCharacter:   clampUint32(item.Range.End.Character),
			Kind:           protocol.FoldingRangeKind(item.Kind),
		}
	})
}

func toProtocolTextEdits(items list.List[TextEdit]) []protocol.TextEdit {
	return mapList(items, func(item TextEdit) protocol.TextEdit {
		return protocol.TextEdit{
			Range:   toProtocolRange(item.Range),
			NewText: item.NewText,
		}
	})
}

func toProtocolCompletionList(items CompletionList) *protocol.CompletionList {
	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        toProtocolCompletionItems(items),
	}
}

func toProtocolCompletionItems(items CompletionList) []protocol.CompletionItem {
	rng := toProtocolRange(items.Range)
	return mapList(items.Items, func(item CompletionItem) protocol.CompletionItem {
		completion := protocol.CompletionItem{
			Label:      item.Label,
			Kind:       protocolCompletionKind(item.Kind),
			Detail:     item.Detail,
			SortText:   item.Label,
			FilterText: item.Label,
			TextEdit: &protocol.TextEdit{
				Range:   rng,
				NewText: item.Label,
			},
		}
		if item.Documentation != "" {
			completion.Documentation = protocol.MarkupContent{
				Kind:  protocol.Markdown,
				Value: item.Documentation,
			}
		}
		return completion
	})
}

func toProtocolDocumentSymbolInterfaces(items list.List[DocumentSymbol]) []any {
	symbols := toProtocolDocumentSymbols(items)
	values := make([]any, 0, len(symbols))
	for index := range symbols {
		values = append(values, symbols[index])
	}
	return values
}

func toProtocolDocumentSymbols(items list.List[DocumentSymbol]) []protocol.DocumentSymbol {
	return mapList(items, func(item DocumentSymbol) protocol.DocumentSymbol {
		return protocol.DocumentSymbol{
			Name:           item.Name,
			Detail:         item.Detail,
			Kind:           protocolSymbolKind(item.Kind),
			Range:          toProtocolRange(item.Range),
			SelectionRange: toProtocolRange(item.SelectionRange),
			Children:       toProtocolDocumentSymbols(item.Children),
		}
	})
}

func toProtocolDiagnostics(items list.List[Diagnostic]) []protocol.Diagnostic {
	return mapList(items, func(item Diagnostic) protocol.Diagnostic {
		diagnostic := protocol.Diagnostic{
			Range:    toProtocolRange(item.Range),
			Severity: protocolSeverity(item.Severity),
			Source:   "plano",
			Message:  item.Message,
		}
		if item.Code != "" {
			diagnostic.Code = item.Code
		}
		if item.Related.Len() > 0 {
			diagnostic.RelatedInformation = toProtocolDiagnosticRelated(item.Related)
		}
		return diagnostic
	})
}

func toProtocolDiagnosticRelated(items list.List[DiagnosticRelatedInformation]) []protocol.DiagnosticRelatedInformation {
	return mapList(items, func(item DiagnosticRelatedInformation) protocol.DiagnosticRelatedInformation {
		return protocol.DiagnosticRelatedInformation{
			Location: protocol.Location{
				URI:   protocol.DocumentURI(item.Location.URI),
				Range: toProtocolRange(item.Location.Range),
			},
			Message: item.Message,
		}
	})
}

func mapList[T any, R any](items list.List[T], mapper func(T) R) []R {
	var out []R
	items.ViewValues(func(values []T) {
		out = make([]R, len(values))
		for index, item := range values {
			out[index] = mapper(item)
		}
	})
	return out
}

func protocolSeverity(severity string) protocol.DiagnosticSeverity {
	switch severity {
	case "warning":
		return protocol.DiagnosticSeverityWarning
	default:
		return protocol.DiagnosticSeverityError
	}
}

var completionKinds = map[CompletionKind]protocol.CompletionItemKind{
	CompletionKeyword:  protocol.CompletionItemKindKeyword,
	CompletionForm:     protocol.CompletionItemKindClass,
	CompletionField:    protocol.CompletionItemKindField,
	CompletionFunction: protocol.CompletionItemKindFunction,
	CompletionAction:   protocol.CompletionItemKindMethod,
	CompletionLocal:    protocol.CompletionItemKindVariable,
	CompletionConst:    protocol.CompletionItemKindConstant,
	CompletionSymbol:   protocol.CompletionItemKindReference,
	CompletionGlobal:   protocol.CompletionItemKindConstant,
	CompletionExprVar:  protocol.CompletionItemKindVariable,
	CompletionExprFunc: protocol.CompletionItemKindFunction,
}

var symbolKinds = map[SymbolKind]protocol.SymbolKind{
	SymbolForm:     protocol.SymbolKindObject,
	SymbolFunction: protocol.SymbolKindFunction,
	SymbolConst:    protocol.SymbolKindConstant,
	SymbolField:    protocol.SymbolKindField,
}

func protocolCompletionKind(kind CompletionKind) protocol.CompletionItemKind {
	if itemKind, ok := completionKinds[kind]; ok {
		return itemKind
	}
	return protocol.CompletionItemKindText
}

func protocolSymbolKind(kind SymbolKind) protocol.SymbolKind {
	if itemKind, ok := symbolKinds[kind]; ok {
		return itemKind
	}
	return protocol.SymbolKindObject
}

func clampUint32(value int) uint32 {
	switch {
	case value <= 0:
		return 0
	case value >= math.MaxUint32:
		return math.MaxUint32
	default:
		return uint32(value)
	}
}
