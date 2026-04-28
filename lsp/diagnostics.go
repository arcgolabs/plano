package lsp

import (
	"path/filepath"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
	"github.com/arcgolabs/plano/diag"
	"github.com/samber/lo"
)

func diagnosticsFromResult(result compiler.Result, sources *mapping.Map[string, []byte]) []Diagnostic {
	return lo.Map(result.Diagnostics, func(item diag.Diagnostic, _ int) Diagnostic {
		return diagnosticFromCompiler(result, sources, item)
	})
}

func diagnosticFromCompiler(result compiler.Result, sources *mapping.Map[string, []byte], item diag.Diagnostic) Diagnostic {
	rng, ok := diagnosticRange(result, sources, item)
	if !ok {
		return Diagnostic{
			Severity: string(item.Severity),
			Message:  item.Message,
		}
	}
	return Diagnostic{
		Severity: string(item.Severity),
		Message:  item.Message,
		Range:    rng,
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
