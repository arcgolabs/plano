package lexer

import "go/token"

type Kind int

const (
	Illegal Kind = iota
	EOF
	Ident
	String
	Int
	Float
	Duration
	Size

	KwImport
	KwConst
	KwLet
	KwFn
	KwReturn
	KwIf
	KwElse
	KwFor
	KwIn
	KwTrue
	KwFalse
	KwNull

	LBrace
	RBrace
	LParen
	RParen
	LBracket
	RBracket
	Comma
	Dot
	Colon

	Assign
	Eq
	NotEq
	GT
	GTE
	LT
	LTE
	Plus
	Minus
	Star
	Slash
	Percent
	Bang
	AndAnd
	OrOr
)

var kindNames = [...]string{
	Illegal:  "illegal",
	EOF:      "eof",
	Ident:    "ident",
	String:   "string",
	Int:      "int",
	Float:    "float",
	Duration: "duration",
	Size:     "size",
	KwImport: "import",
	KwConst:  "const",
	KwLet:    "let",
	KwFn:     "fn",
	KwReturn: "return",
	KwIf:     "if",
	KwElse:   "else",
	KwFor:    "for",
	KwIn:     "in",
	KwTrue:   "true",
	KwFalse:  "false",
	KwNull:   "null",
	LBrace:   "{",
	RBrace:   "}",
	LParen:   "(",
	RParen:   ")",
	LBracket: "[",
	RBracket: "]",
	Comma:    ",",
	Dot:      ".",
	Colon:    ":",
	Assign:   "=",
	Eq:       "==",
	NotEq:    "!=",
	GT:       ">",
	GTE:      ">=",
	LT:       "<",
	LTE:      "<=",
	Plus:     "+",
	Minus:    "-",
	Star:     "*",
	Slash:    "/",
	Percent:  "%",
	Bang:     "!",
	AndAnd:   "&&",
	OrOr:     "||",
}

func (k Kind) String() string {
	if int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return "unknown"
}

type Token struct {
	Kind Kind
	Pos  token.Pos
	End  token.Pos
	Text string
}

var keywords = map[string]Kind{
	"import": KwImport,
	"const":  KwConst,
	"let":    KwLet,
	"fn":     KwFn,
	"return": KwReturn,
	"if":     KwIf,
	"else":   KwElse,
	"for":    KwFor,
	"in":     KwIn,
	"true":   KwTrue,
	"false":  KwFalse,
	"null":   KwNull,
}
