package parser

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/internal/lexer"
)

func (p *Parser) parseArrayExpr() ast.Expr {
	lbrack := p.expect(lexer.LBracket, "expected [")
	array := &ast.ArrayExpr{Lbrack: lbrack.Pos}
	array.Elements = p.parseExprList(lexer.RBracket)
	rbrack := p.expect(lexer.RBracket, "expected ]")
	array.Rbrack = rbrack.End
	return array
}

func (p *Parser) parseObjectExpr() ast.Expr {
	lbrace := p.expect(lexer.LBrace, "expected {")
	object := &ast.ObjectExpr{Lbrace: lbrace.Pos}
	for p.cur.Kind != lexer.RBrace && p.cur.Kind != lexer.EOF {
		p.parseObjectEntry(object)
		if p.cur.Kind != lexer.Comma {
			break
		}
		p.advance()
		if p.cur.Kind == lexer.RBrace {
			break
		}
	}
	rbrace := p.expect(lexer.RBrace, "expected }")
	object.Rbrace = rbrace.End
	return object
}

func (p *Parser) parseObjectEntry(object *ast.ObjectExpr) {
	key := p.parseIdent()
	p.expect(lexer.Assign, "expected =")
	object.Entries = append(object.Entries, &ast.ObjectEntry{
		Key:   key,
		Value: p.parseExpr(),
	})
}
