package lsp

import (
	"go/token"
	"path/filepath"

	"github.com/arcgolabs/collectionx/interval"
	"github.com/arcgolabs/collectionx/mapping"
)

func (s Snapshot) tokenPos(pos Position) (token.Pos, bool) {
	file, fileOK := s.fileForPath(s.Path)
	src, ok := s.source(s.Path)
	if !fileOK || !ok || file == nil {
		return token.NoPos, false
	}
	offset, ok := offsetFromPosition(src, pos)
	if !ok {
		return token.NoPos, false
	}
	if offset > file.Size() {
		offset = file.Size()
	}
	return file.Pos(offset), true
}

func (s Snapshot) locationForSpan(pos, end token.Pos) (Location, bool) {
	path, rng, ok := s.spanLocation(pos, end)
	if !ok {
		return Location{}, false
	}
	return Location{
		URI:   s.uriForPath(path),
		Range: rng,
	}, true
}

func (s Snapshot) rangeForSpan(pos, end token.Pos) (Range, bool) {
	_, rng, ok := s.spanLocation(pos, end)
	return rng, ok
}

func (s Snapshot) spanLocation(pos, end token.Pos) (string, Range, bool) {
	if !pos.IsValid() || s.Result.FileSet == nil {
		return "", Range{}, false
	}
	span, ok := s.fileSpanAt(pos)
	if !ok || span.file == nil {
		return "", Range{}, false
	}
	if !end.IsValid() || end < pos {
		end = pos
	}
	path := span.path
	src, ok := s.source(path)
	if !ok {
		return "", Range{}, false
	}
	file := span.file
	startOffset := file.Offset(pos)
	endOffset := min(file.Offset(end), len(src))
	endOffset = max(endOffset, startOffset)
	return path, Range{
		Start: positionFromFileOffset(src, file, startOffset),
		End:   positionFromFileOffset(src, file, endOffset),
	}, true
}

func (s Snapshot) source(path string) ([]byte, bool) {
	clean := filepath.Clean(path)
	if src, ok := s.sources.Get(clean); ok {
		return src, true
	}
	if doc, ok := s.documents.Get(clean); ok {
		return doc.Text, true
	}
	return nil, false
}

func (s Snapshot) uriForPath(path string) string {
	clean := filepath.Clean(path)
	if doc, ok := s.documents.Get(clean); ok && doc.URI != "" {
		return doc.URI
	}
	return FileURI(clean)
}

func (s Snapshot) fileForPath(path string) (*token.File, bool) {
	clean := filepath.Clean(path)
	if s.files != nil {
		if file, ok := s.files.Get(clean); ok {
			return file, true
		}
	}
	file := fileForPath(s.Result.FileSet, clean)
	if file == nil {
		return nil, false
	}
	return file, true
}

func (s Snapshot) fileSpanAt(pos token.Pos) (fileSpan, bool) {
	if s.fileSpans != nil {
		if span, ok := s.fileSpans.Get(int(pos)); ok {
			return span, true
		}
	}
	file := s.Result.FileSet.File(pos)
	if file == nil {
		return fileSpan{}, false
	}
	return fileSpan{
		path: filepath.Clean(file.Name()),
		file: file,
	}, true
}

func buildFileSetIndex(fset *token.FileSet) (*mapping.Map[string, *token.File], *interval.RangeMap[int, fileSpan]) {
	files := mapping.NewMap[string, *token.File]()
	spans := interval.NewRangeMap[int, fileSpan]()
	if fset == nil {
		return files, spans
	}
	fset.Iterate(func(file *token.File) bool {
		if file == nil {
			return true
		}
		path := filepath.Clean(file.Name())
		files.Set(path, file)
		start := file.Base()
		end := file.Base() + file.Size() + 1
		spans.Put(start, end, fileSpan{
			path: path,
			file: file,
		})
		return true
	})
	return files, spans
}
