package lsp

import (
	"go/token"
	"unicode/utf16"
	"unicode/utf8"
)

func positionFromOffset(src []byte, offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	line := 0
	character := 0
	for idx := 0; idx < offset; {
		if src[idx] == '\n' {
			line++
			character = 0
			idx++
			continue
		}
		r, size := utf8.DecodeRune(src[idx:])
		character += utf16Width(r)
		idx += size
	}
	return Position{
		Line:      line,
		Character: character,
	}
}

func positionFromFileOffset(src []byte, file *token.File, offset int) Position {
	if file == nil {
		return positionFromOffset(src, offset)
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(src) {
		offset = len(src)
	}
	pos := file.Pos(offset)
	tokenPosition := file.Position(pos)
	if tokenPosition.Line <= 0 {
		return positionFromOffset(src, offset)
	}
	lineStart := file.Offset(file.LineStart(tokenPosition.Line))
	if lineStart < 0 || lineStart > offset || lineStart > len(src) {
		return positionFromOffset(src, offset)
	}
	return Position{
		Line:      tokenPosition.Line - 1,
		Character: utf16WidthBetween(src, lineStart, offset),
	}
}

func offsetFromPosition(src []byte, pos Position) (int, bool) {
	lineStart, ok := lineStartOffset(src, pos.Line)
	if !ok {
		return 0, false
	}
	return advanceOffsetByUTF16(src, lineStart, lineEndOffset(src, lineStart), pos.Character), true
}

func utf16WidthBetween(src []byte, start, end int) int {
	if start < 0 {
		start = 0
	}
	if end > len(src) {
		end = len(src)
	}
	if end < start {
		end = start
	}
	width := 0
	for offset := start; offset < end; {
		r, size := utf8.DecodeRune(src[offset:end])
		width += utf16Width(r)
		offset += size
	}
	return width
}

func utf16Width(r rune) int {
	if r == utf8.RuneError {
		return 1
	}
	return len(utf16.Encode([]rune{r}))
}

func lineStartOffset(src []byte, line int) (int, bool) {
	if line < 0 {
		return 0, false
	}
	offset := 0
	for range line {
		next := lineEndOffset(src, offset)
		if next >= len(src) {
			return 0, false
		}
		offset = next + 1
	}
	return offset, true
}

func lineEndOffset(src []byte, start int) int {
	end := start
	for end < len(src) && src[end] != '\n' {
		_, size := utf8.DecodeRune(src[end:])
		end += size
	}
	return end
}

func advanceOffsetByUTF16(src []byte, start, end, character int) int {
	if character <= 0 {
		return start
	}
	offset := start
	width := 0
	for offset < end {
		if width >= character {
			return offset
		}
		r, size := utf8.DecodeRune(src[offset:])
		nextWidth := width + utf16Width(r)
		if nextWidth > character {
			return offset
		}
		width = nextWidth
		offset += size
	}
	return end
}
