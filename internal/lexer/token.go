//nolint:cyclop,gocyclo,funlen // Token names are intentionally listed exhaustively for stable diagnostics.
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

func (k Kind) String() string {
	switch k {
	case Illegal:
		return "illegal"
	case EOF:
		return "eof"
	case Ident:
		return "ident"
	case String:
		return "string"
	case Int:
		return "int"
	case Float:
		return "float"
	case Duration:
		return "duration"
	case Size:
		return "size"
	case KwImport:
		return "import"
	case KwConst:
		return "const"
	case KwLet:
		return "let"
	case KwFn:
		return "fn"
	case KwReturn:
		return "return"
	case KwIf:
		return "if"
	case KwElse:
		return "else"
	case KwFor:
		return "for"
	case KwIn:
		return "in"
	case KwTrue:
		return "true"
	case KwFalse:
		return "false"
	case KwNull:
		return "null"
	case LBrace:
		return "{"
	case RBrace:
		return "}"
	case LParen:
		return "("
	case RParen:
		return ")"
	case LBracket:
		return "["
	case RBracket:
		return "]"
	case Comma:
		return ","
	case Dot:
		return "."
	case Colon:
		return ":"
	case Assign:
		return "="
	case Eq:
		return "=="
	case NotEq:
		return "!="
	case GT:
		return ">"
	case GTE:
		return ">="
	case LT:
		return "<"
	case LTE:
		return "<="
	case Plus:
		return "+"
	case Minus:
		return "-"
	case Star:
		return "*"
	case Slash:
		return "/"
	case Percent:
		return "%"
	case Bang:
		return "!"
	case AndAnd:
		return "&&"
	case OrOr:
		return "||"
	default:
		return "unknown"
	}
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
