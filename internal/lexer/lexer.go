// Package lexer tokenizes plano source text.
//
//nolint:cyclop,gocognit,gocyclo,funlen // Lexing keeps the token-state machine in one place.
package lexer

import (
	"go/token"
	"strings"
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

	switch ch {
	case '{':
		l.offset++
		return l.token(LBrace, start, l.offset)
	case '}':
		l.offset++
		return l.token(RBrace, start, l.offset)
	case '(':
		l.offset++
		return l.token(LParen, start, l.offset)
	case ')':
		l.offset++
		return l.token(RParen, start, l.offset)
	case '[':
		l.offset++
		return l.token(LBracket, start, l.offset)
	case ']':
		l.offset++
		return l.token(RBracket, start, l.offset)
	case ',':
		l.offset++
		return l.token(Comma, start, l.offset)
	case '.':
		l.offset++
		return l.token(Dot, start, l.offset)
	case ':':
		l.offset++
		return l.token(Colon, start, l.offset)
	case '+':
		l.offset++
		return l.token(Plus, start, l.offset)
	case '-':
		l.offset++
		return l.token(Minus, start, l.offset)
	case '*':
		l.offset++
		return l.token(Star, start, l.offset)
	case '%':
		l.offset++
		return l.token(Percent, start, l.offset)
	case '!':
		if l.match('=') {
			return l.token(NotEq, start, l.offset)
		}
		l.offset++
		return l.token(Bang, start, l.offset)
	case '=':
		if l.match('=') {
			return l.token(Eq, start, l.offset)
		}
		l.offset++
		return l.token(Assign, start, l.offset)
	case '>':
		if l.match('=') {
			return l.token(GTE, start, l.offset)
		}
		l.offset++
		return l.token(GT, start, l.offset)
	case '<':
		if l.match('=') {
			return l.token(LTE, start, l.offset)
		}
		l.offset++
		return l.token(LT, start, l.offset)
	case '&':
		if l.match('&') {
			return l.token(AndAnd, start, l.offset)
		}
	case '|':
		if l.match('|') {
			return l.token(OrOr, start, l.offset)
		}
	case '/':
		l.offset++
		return l.token(Slash, start, l.offset)
	case '"':
		return l.scanString()
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
		for l.offset < len(l.src) && unicode.IsSpace(rune(l.src[l.offset])) {
			l.offset++
		}
		if l.offset+1 >= len(l.src) || l.src[l.offset] != '/' {
			return
		}
		switch l.src[l.offset+1] {
		case '/':
			l.offset += 2
			for l.offset < len(l.src) && l.src[l.offset] != '\n' {
				l.offset++
			}
		case '*':
			start := l.offset
			l.offset += 2
			for l.offset+1 < len(l.src) && (l.src[l.offset] != '*' || l.src[l.offset+1] != '/') {
				l.offset++
			}
			if l.offset+1 >= len(l.src) {
				l.diags.AddError(l.file.Pos(start), l.file.Pos(len(l.src)), "unterminated block comment")
				l.offset = len(l.src)
				return
			}
			l.offset += 2
		default:
			return
		}
	}
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
			raw := string(l.src[start+1 : l.offset-1])
			return Token{
				Kind: String,
				Pos:  l.file.Pos(start),
				End:  l.file.Pos(l.offset),
				Text: unescape(raw),
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
	for l.offset < len(l.src) && isDigit(l.src[l.offset]) {
		l.offset++
	}
	kind := Int
	if l.offset < len(l.src) && l.src[l.offset] == '.' && l.offset+1 < len(l.src) && isDigit(l.src[l.offset+1]) {
		kind = Float
		l.offset++
		for l.offset < len(l.src) && isDigit(l.src[l.offset]) {
			l.offset++
		}
		return l.token(kind, start, l.offset)
	}

	unitStart := l.offset
	for l.offset < len(l.src) && unicode.IsLetter(rune(l.src[l.offset])) {
		l.offset++
	}
	if unitStart == l.offset {
		return l.token(kind, start, l.offset)
	}

	unit := string(l.src[unitStart:l.offset])
	switch unit {
	case "ms", "s", "m", "h":
		return l.token(Duration, start, l.offset)
	case "B", "Ki", "Mi", "Gi", "Ti":
		return l.token(Size, start, l.offset)
	default:
		l.diags.AddError(l.file.Pos(start), l.file.Pos(l.offset), "invalid numeric suffix")
		return l.token(Illegal, start, l.offset)
	}
}

func (l *Lexer) match(next byte) bool {
	if l.offset+1 < len(l.src) && l.src[l.offset+1] == next {
		l.offset += 2
		return true
	}
	return false
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentStart(ch byte) bool {
	return ch == '_' || unicode.IsLetter(rune(ch))
}

func isIdentContinue(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}

func unescape(s string) string {
	replacer := strings.NewReplacer(
		`\\`, `\`,
		`\n`, "\n",
		`\r`, "\r",
		`\t`, "\t",
		`\"`, `"`,
	)
	return replacer.Replace(s)
}
