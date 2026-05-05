package lsp

import (
	"go/token"
	"testing"
)

func TestPositionFromFileOffsetPreservesUTF16Columns(t *testing.T) {
	src := []byte("a😀b\nçd")
	file := token.NewFileSet().AddFile("unicode.plano", -1, len(src))
	file.AddLine(len([]byte("a😀b\n")))

	afterEmoji := len([]byte("a😀"))
	got := positionFromFileOffset(src, file, afterEmoji)
	if got != (Position{Line: 0, Character: 3}) {
		t.Fatalf("position after emoji = %#v", got)
	}

	secondLine := len([]byte("a😀b\nç"))
	got = positionFromFileOffset(src, file, secondLine)
	if got != (Position{Line: 1, Character: 1}) {
		t.Fatalf("position on second line = %#v", got)
	}
}
