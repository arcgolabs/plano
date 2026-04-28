package lsp

import (
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

func offsetFromPosition(src []byte, pos Position) (int, bool) {
	lineStart, ok := lineStartOffset(src, pos.Line)
	if !ok {
		return 0, false
	}
	return advanceOffsetByUTF16(src, lineStart, lineEndOffset(src, lineStart), pos.Character), true
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
