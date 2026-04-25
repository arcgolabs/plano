package lexer

import "unicode"

var singleRuneTokens = map[byte]Kind{
	'{': LBrace,
	'}': RBrace,
	'(': LParen,
	')': RParen,
	'[': LBracket,
	']': RBracket,
	',': Comma,
	'.': Dot,
	':': Colon,
	'+': Plus,
	'-': Minus,
	'*': Star,
	'%': Percent,
	'/': Slash,
}

func isDurationUnit(unit string) bool {
	return unit == "ms" || unit == "s" || unit == "m" || unit == "h"
}

func isSizeUnit(unit string) bool {
	return unit == "B" || unit == "Ki" || unit == "Mi" || unit == "Gi" || unit == "Ti"
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
