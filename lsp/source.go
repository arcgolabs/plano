package lsp

import (
	"go/token"
	"path/filepath"
)

func (s Snapshot) tokenPos(pos Position) (token.Pos, bool) {
	file := fileForPath(s.Result.FileSet, s.Path)
	src, ok := s.source(s.Path)
	if !ok || file == nil {
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
	file := s.Result.FileSet.File(pos)
	if file == nil {
		return "", Range{}, false
	}
	if !end.IsValid() || end < pos {
		end = pos
	}
	path := filepath.Clean(file.Name())
	src, ok := s.source(path)
	if !ok {
		return "", Range{}, false
	}
	startOffset := file.Offset(pos)
	endOffset := min(file.Offset(end), len(src))
	endOffset = max(endOffset, startOffset)
	return path, Range{
		Start: positionFromOffset(src, startOffset),
		End:   positionFromOffset(src, endOffset),
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
