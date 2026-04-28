package lsp

import (
	"math"

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

func toProtocolDiagnostics(items []Diagnostic) []protocol.Diagnostic {
	out := make([]protocol.Diagnostic, 0, len(items))
	for _, item := range items {
		out = append(out, protocol.Diagnostic{
			Range:    toProtocolRange(item.Range),
			Severity: protocolSeverity(item.Severity),
			Source:   "plano",
			Message:  item.Message,
		})
	}
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
