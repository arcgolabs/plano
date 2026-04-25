package parser

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/internal/lexer"
)

var precedenceByKind = [...]int{
	lexer.OrOr:    1,
	lexer.AndAnd:  2,
	lexer.Eq:      3,
	lexer.NotEq:   3,
	lexer.GT:      4,
	lexer.GTE:     4,
	lexer.LT:      4,
	lexer.LTE:     4,
	lexer.Plus:    5,
	lexer.Minus:   5,
	lexer.Star:    6,
	lexer.Slash:   6,
	lexer.Percent: 6,
}

func (p *Parser) parseQualifiedIdent() *ast.QualifiedIdent {
	qid := &ast.QualifiedIdent{}
	qid.Parts = append(qid.Parts, p.parseIdent())
	for p.cur.Kind == lexer.Dot {
		p.advance()
		qid.Parts = append(qid.Parts, p.parseIdent())
	}
	return qid
}

func (p *Parser) parseIdent() *ast.Ident {
	if p.cur.Kind != lexer.Ident {
		p.error(p.cur.Pos, p.cur.End, "expected identifier")
		bad := p.cur
		if p.cur.Kind != lexer.EOF {
			p.advance()
		}
		return &ast.Ident{Name: bad.Text, Start: bad.Pos, Finish: bad.End}
	}
	tok := p.cur
	p.advance()
	return &ast.Ident{Name: tok.Text, Start: tok.Pos, Finish: tok.End}
}

func (p *Parser) parseStringLiteral() *ast.StringLiteral {
	if p.cur.Kind != lexer.String {
		p.error(p.cur.Pos, p.cur.End, "expected string literal")
		bad := p.cur
		if p.cur.Kind != lexer.EOF {
			p.advance()
		}
		return &ast.StringLiteral{Start: bad.Pos, Finish: bad.End, Value: bad.Text}
	}
	tok := p.cur
	p.advance()
	return &ast.StringLiteral{Start: tok.Pos, Finish: tok.End, Value: tok.Text}
}

func (p *Parser) parseExprList(end lexer.Kind) []ast.Expr {
	items := make([]ast.Expr, 0)
	for p.cur.Kind != end && p.cur.Kind != lexer.EOF {
		items = append(items, p.parseExpr())
		if p.cur.Kind != lexer.Comma {
			break
		}
		p.advance()
		if p.cur.Kind == end {
			break
		}
	}
	return items
}

func precedence(kind lexer.Kind) int {
	if int(kind) >= len(precedenceByKind) {
		return 0
	}
	return precedenceByKind[kind]
}
