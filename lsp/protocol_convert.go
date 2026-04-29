package lsp

import (
	"math"

	"github.com/arcgolabs/collectionx/list"
	"github.com/samber/lo"
	"go.lsp.dev/protocol"
)

func fromProtocolPosition(pos protocol.Position) Position {
	return Position{
		Line:      int(pos.Line),
		Character: int(pos.Character),
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
	return lo.Map(items.Values(), func(item Location, _ int) protocol.Location {
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

func toProtocolCompletionList(items CompletionList) *protocol.CompletionList {
	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        toProtocolCompletionItems(items),
	}
}

func toProtocolCompletionItems(items CompletionList) []protocol.CompletionItem {
	rng := toProtocolRange(items.Range)
	return lo.Map(items.Items.Values(), func(item CompletionItem, _ int) protocol.CompletionItem {
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
	return lo.Map(items.Values(), func(item DocumentSymbol, _ int) protocol.DocumentSymbol {
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
	return lo.Map(items.Values(), func(item Diagnostic, _ int) protocol.Diagnostic {
		return protocol.Diagnostic{
			Range:    toProtocolRange(item.Range),
			Severity: protocolSeverity(item.Severity),
			Source:   "plano",
			Message:  item.Message,
		}
	})
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
