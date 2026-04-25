// Package lexer tokenizes plano source text.
package lexer

import (
	"go/token"
	"strconv"
	"unicode"

	"github.com/arcgolabs/plano/diag"
)

type Lexer struct {
	file   *token.File
	src    []byte
	offset int
	diags  diag.Diagnostics
}

func New(file *token.File, src []byte) *Lexer {
	return &Lexer{
		file: file,
		src:  src,
	}
}

func (l *Lexer) Diagnostics() diag.Diagnostics {
	return l.diags
}

func (l *Lexer) Next() Token {
	l.skipSpaceAndComments()
	if l.offset >= len(l.src) {
		pos := l.file.Pos(len(l.src))
		return Token{Kind: EOF, Pos: pos, End: pos}
	}

	ch := l.src[l.offset]
	start := l.offset

	if tok, ok := l.scanPunctuation(start, ch); ok {
		return tok
	}

	if isIdentStart(ch) {
		return l.scanIdent()
	}
	if isDigit(ch) {
		return l.scanNumber()
	}

	l.offset++
	l.diags.AddError(l.file.Pos(start), l.file.Pos(l.offset), "invalid character")
	return l.token(Illegal, start, l.offset)
}

func (l *Lexer) scanPunctuation(start int, ch byte) (Token, bool) {
	if tok, ok := l.scanSingleRuneToken(start, ch); ok {
		return tok, true
	}
	switch ch {
	case '!':
		return l.scanPairedToken(start, NotEq, Bang)
	case '=':
		return l.scanPairedToken(start, Eq, Assign)
	case '>':
		return l.scanPairedToken(start, GTE, GT)
	case '<':
		return l.scanPairedToken(start, LTE, LT)
	case '&':
		return l.scanLogicalToken(start, '&', AndAnd)
	case '|':
		return l.scanLogicalToken(start, '|', OrOr)
	case '"':
		return l.scanString(), true
	default:
		return Token{}, false
	}
}

func (l *Lexer) scanSingleRuneToken(start int, ch byte) (Token, bool) {
	kind, ok := singleRuneTokens[ch]
	if !ok {
		return Token{}, false
	}
	return l.advanceToken(kind, start), true
}

func (l *Lexer) scanPairedToken(start int, paired, single Kind) (Token, bool) {
	if l.match('=') {
		return l.token(paired, start, l.offset), true
	}
	return l.advanceToken(single, start), true
}

func (l *Lexer) scanLogicalToken(start int, next byte, kind Kind) (Token, bool) {
	if l.match(next) {
		return l.token(kind, start, l.offset), true
	}
	return Token{}, false
}

func (l *Lexer) advanceToken(kind Kind, start int) Token {
	l.offset++
	return l.token(kind, start, l.offset)
}

func (l *Lexer) token(kind Kind, start, end int) Token {
	return Token{
		Kind: kind,
		Pos:  l.file.Pos(start),
		End:  l.file.Pos(end),
		Text: string(l.src[start:end]),
	}
}

func (l *Lexer) skipSpaceAndComments() {
	for {
		l.skipWhitespace()
		if !l.skipComment() {
			return
		}
	}
}

func (l *Lexer) skipWhitespace() {
	for l.offset < len(l.src) && unicode.IsSpace(rune(l.src[l.offset])) {
		l.offset++
	}
}

func (l *Lexer) skipComment() bool {
	if l.offset+1 >= len(l.src) || l.src[l.offset] != '/' {
		return false
	}
	if l.src[l.offset+1] == '/' {
		l.skipLineComment()
		return true
	}
	if l.src[l.offset+1] == '*' {
		return l.skipBlockComment()
	}
	return false
}

func (l *Lexer) skipLineComment() {
	l.offset += 2
	for l.offset < len(l.src) && l.src[l.offset] != '\n' {
		l.offset++
	}
}

func (l *Lexer) skipBlockComment() bool {
	start := l.offset
	l.offset += 2
	for l.offset+1 < len(l.src) && (l.src[l.offset] != '*' || l.src[l.offset+1] != '/') {
		l.offset++
	}
	if l.offset+1 >= len(l.src) {
		l.diags.AddError(l.file.Pos(start), l.file.Pos(len(l.src)), "unterminated block comment")
		l.offset = len(l.src)
		return false
	}
	l.offset += 2
	return true
}

func (l *Lexer) scanIdent() Token {
	start := l.offset
	l.offset++
	for l.offset < len(l.src) && isIdentContinue(l.src[l.offset]) {
		l.offset++
	}
	kind, ok := keywords[string(l.src[start:l.offset])]
	if ok {
		return l.token(kind, start, l.offset)
	}
	return l.token(Ident, start, l.offset)
}

func (l *Lexer) scanString() Token {
	start := l.offset
	l.offset++
	for l.offset < len(l.src) {
		ch := l.src[l.offset]
		if ch == '"' {
			l.offset++
			raw := string(l.src[start:l.offset])
			text, err := strconv.Unquote(raw)
			if err != nil {
				l.diags.AddError(l.file.Pos(start), l.file.Pos(l.offset), "invalid string literal")
				text = string(l.src[start+1 : l.offset-1])
			}
			return Token{
				Kind: String,
				Pos:  l.file.Pos(start),
				End:  l.file.Pos(l.offset),
				Text: text,
			}
		}
		if ch == '\\' {
			l.offset += 2
			continue
		}
		l.offset++
	}
	l.diags.AddError(l.file.Pos(start), l.file.Pos(len(l.src)), "unterminated string literal")
	return Token{
		Kind: String,
		Pos:  l.file.Pos(start),
		End:  l.file.Pos(len(l.src)),
		Text: string(l.src[start+1:]),
	}
}

func (l *Lexer) scanNumber() Token {
	start := l.offset
	l.scanDigits()
	if l.hasFraction() {
		l.offset++
		l.scanDigits()
		return l.token(Float, start, l.offset)
	}
	unitStart := l.offset
	l.scanLetters()
	if unitStart == l.offset {
		return l.token(Int, start, l.offset)
	}
	return l.tokenWithNumericSuffix(start, unitStart)
}

func (l *Lexer) scanDigits() {
	for l.offset < len(l.src) && isDigit(l.src[l.offset]) {
		l.offset++
	}
}

func (l *Lexer) hasFraction() bool {
	return l.offset < len(l.src) && l.src[l.offset] == '.' && l.offset+1 < len(l.src) && isDigit(l.src[l.offset+1])
}

func (l *Lexer) scanLetters() {
	for l.offset < len(l.src) && unicode.IsLetter(rune(l.src[l.offset])) {
		l.offset++
	}
}

func (l *Lexer) tokenWithNumericSuffix(start, unitStart int) Token {
	unit := string(l.src[unitStart:l.offset])
	if isDurationUnit(unit) {
		return l.token(Duration, start, l.offset)
	}
	if isSizeUnit(unit) {
		return l.token(Size, start, l.offset)
	}
	l.diags.AddError(l.file.Pos(start), l.file.Pos(l.offset), "invalid numeric suffix")
	return l.token(Illegal, start, l.offset)
}

func (l *Lexer) match(next byte) bool {
	if l.offset+1 < len(l.src) && l.src[l.offset+1] == next {
		l.offset += 2
		return true
	}
	return false
}
