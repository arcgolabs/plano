package lsp

import (
	"path/filepath"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/diag"
)

func diagnosticsFromResult(result compiler.Result, sources *mapping.Map[string, []byte]) list.List[Diagnostic] {
	out := list.NewListWithCapacity[Diagnostic](len(result.Diagnostics))
	for index := range len(result.Diagnostics) {
		item := result.Diagnostics[index]
		out.Add(diagnosticFromCompiler(result, sources, item))
	}
	return *out
}

func diagnosticFromCompiler(result compiler.Result, sources *mapping.Map[string, []byte], item diag.Diagnostic) Diagnostic {
	rng, ok := diagnosticRange(result, sources, item)
	related := diagnosticRelated(result, sources, item.Related)
	if !ok {
		return Diagnostic{
			Severity: string(item.Severity),
			Code:     string(item.Code),
			Message:  item.Message,
			Related:  related,
		}
	}
	return Diagnostic{
		Severity: string(item.Severity),
		Code:     string(item.Code),
		Message:  item.Message,
		Range:    rng,
		Related:  related,
	}
}

func diagnosticRange(result compiler.Result, sources *mapping.Map[string, []byte], item diag.Diagnostic) (Range, bool) {
	if !item.Pos.IsValid() || result.FileSet == nil {
		return Range{}, false
	}
	file := result.FileSet.File(item.Pos)
	if file == nil {
		return Range{}, false
	}
	path := filepath.Clean(file.Name())
	src, ok := sources.Get(path)
	if !ok {
		return Range{}, false
	}
	end := item.End
	if !end.IsValid() || end < item.Pos {
		end = item.Pos
	}
	startOffset := file.Offset(item.Pos)
	endOffset := min(file.Offset(end), len(src))
	endOffset = max(endOffset, startOffset)
	return Range{
		Start: positionFromOffset(src, startOffset),
		End:   positionFromOffset(src, endOffset),
	}, true
}

func diagnosticRelated(
	result compiler.Result,
	sources *mapping.Map[string, []byte],
	items list.List[diag.RelatedInformation],
) list.List[DiagnosticRelatedInformation] {
	out := list.NewListWithCapacity[DiagnosticRelatedInformation](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		location, ok := diagnosticRelatedLocation(result, sources, item)
		if !ok {
			continue
		}
		out.Add(DiagnosticRelatedInformation{
			Message:  item.Message,
			Location: location,
		})
	}
	return *out
}

func diagnosticRelatedLocation(
	result compiler.Result,
	sources *mapping.Map[string, []byte],
	item diag.RelatedInformation,
) (Location, bool) {
	rng, ok := diagnosticRange(result, sources, diag.Diagnostic{Pos: item.Pos, End: item.End})
	if !ok || !item.Pos.IsValid() || result.FileSet == nil {
		return Location{}, false
	}
	file := result.FileSet.File(item.Pos)
	if file == nil {
		return Location{}, false
	}
	path := filepath.Clean(file.Name())
	return Location{
		URI:   FileURI(path),
		Range: rng,
	}, true
}
