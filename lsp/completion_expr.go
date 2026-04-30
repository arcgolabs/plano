package lsp

import (
	"go/token"

	"github.com/arcgolabs/plano/internal/lexer"
)

func (s Snapshot) exprLangCompletionContext(pos Position) (completionContext, bool) {
	target, src, offset, ok := s.exprLangSourcePosition(pos)
	if !ok {
		return completionContext{}, false
	}
	file, ok := s.fileForPath(s.Path)
	if !ok || file == nil {
		return completionContext{}, false
	}
	stringTok, ok := exprLangStringTokenAt(file, src, target)
	if !ok {
		return completionContext{}, false
	}
	start, end := exprLangStringCompletionBounds(file, src, stringTok, offset)
	if start < 0 {
		return completionContext{}, false
	}
	return completionContext{
		target: target,
		offset: offset,
		prefix: string(src[start:offset]),
		rng: Range{
			Start: positionFromOffset(src, start),
			End:   positionFromOffset(src, end),
		},
	}, true
}

func (s Snapshot) exprLangSourcePosition(pos Position) (token.Pos, []byte, int, bool) {
	src, ok := s.source(s.Path)
	if !ok {
		return token.NoPos, nil, 0, false
	}
	offset, ok := offsetFromPosition(src, pos)
	if !ok {
		return token.NoPos, nil, 0, false
	}
	target, ok := s.tokenPos(pos)
	if !ok {
		return token.NoPos, nil, 0, false
	}
	return target, src, offset, true
}

func exprLangStringTokenAt(file *token.File, src []byte, target token.Pos) (lexer.Token, bool) {
	var prev lexer.Token
	var prevPrev lexer.Token
	scanner := lexer.New(file, src)
	for tok := scanner.Next(); tok.Kind != lexer.EOF; tok = scanner.Next() {
		if tok.Kind == lexer.String && tok.Pos < target && target <= tok.End && isExprLangFirstArg(prevPrev, prev) {
			return tok, true
		}
		if tok.Pos > target {
			return lexer.Token{}, false
		}
		prevPrev = prev
		prev = tok
	}
	return lexer.Token{}, false
}

func isExprLangFirstArg(fn, open lexer.Token) bool {
	if open.Kind != lexer.LParen || fn.Kind != lexer.Ident {
		return false
	}
	return fn.Text == "expr" || fn.Text == "expr_eval"
}

func exprLangStringCompletionBounds(file *token.File, src []byte, tok lexer.Token, offset int) (int, int) {
	contentStart := file.Offset(tok.Pos) + 1
	contentEnd := file.Offset(tok.End)
	if contentEnd > contentStart && contentEnd <= len(src) && src[contentEnd-1] == '"' {
		contentEnd--
	}
	if offset < contentStart || offset > contentEnd {
		return -1, -1
	}
	start, end := completionBounds(src, offset)
	if start < contentStart {
		start = contentStart
	}
	if end > contentEnd {
		end = contentEnd
	}
	return start, end
}
