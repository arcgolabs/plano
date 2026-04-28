package parser

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/internal/lexer"
)

func (p *Parser) parseImportDecl() *ast.ImportDecl {
	start := p.expect(lexer.KwImport, "expected import")
	path := p.parseStringLiteral()
	return &ast.ImportDecl{
		Import: start.Pos,
		Path:   path,
	}
}

func (p *Parser) parseConstDecl() *ast.ConstDecl {
	start := p.expect(lexer.KwConst, "expected const")
	name, typeExpr, value := p.parseBoundValue()
	return &ast.ConstDecl{
		Const: start.Pos,
		Name:  name,
		Type:  typeExpr,
		Value: value,
	}
}

func (p *Parser) parseLetDecl() *ast.LetDecl {
	start := p.expect(lexer.KwLet, "expected let")
	name, typeExpr, value := p.parseBoundValue()
	return &ast.LetDecl{
		Let:   start.Pos,
		Name:  name,
		Type:  typeExpr,
		Value: value,
	}
}

func (p *Parser) parseFnDecl() *ast.FnDecl {
	start := p.expect(lexer.KwFn, "expected fn")
	name := p.parseIdent()
	params := p.parseParams()
	var result ast.TypeExpr
	if p.cur.Kind == lexer.Colon {
		p.advance()
		result = p.parseType()
	}
	return &ast.FnDecl{
		Fn:     start.Pos,
		Name:   name,
		Params: params,
		Result: result,
		Body:   p.parseBlock(),
	}
}

func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	start := p.expect(lexer.KwReturn, "expected return")
	return &ast.ReturnStmt{
		Return: start.Pos,
		Value:  p.parseExpr(),
	}
}

func (p *Parser) parseBreakStmt() *ast.BreakStmt {
	start := p.expect(lexer.KwBreak, "expected break")
	return &ast.BreakStmt{Break: start.Pos}
}

func (p *Parser) parseContinueStmt() *ast.ContinueStmt {
	start := p.expect(lexer.KwContinue, "expected continue")
	return &ast.ContinueStmt{Continue: start.Pos}
}

func (p *Parser) parseIfStmt() *ast.IfStmt {
	start := p.expect(lexer.KwIf, "expected if")
	condition := p.parseExpr()
	thenBlock := p.parseBlock()
	var elseBlock *ast.Block
	if p.cur.Kind == lexer.KwElse {
		p.advance()
		if p.cur.Kind == lexer.KwIf {
			nested := p.parseIfStmt()
			elseBlock = &ast.Block{
				Lbrace: nested.Pos(),
				Rbrace: nested.End(),
				Items:  []ast.FormItem{nested},
			}
		} else {
			elseBlock = p.parseBlock()
		}
	}
	return &ast.IfStmt{
		If:        start.Pos,
		Condition: condition,
		Then:      thenBlock,
		Else:      elseBlock,
	}
}

func (p *Parser) parseForStmt() *ast.ForStmt {
	start := p.expect(lexer.KwFor, "expected for")
	first := p.parseIdent()
	var index *ast.Ident
	name := first
	if p.cur.Kind == lexer.Comma {
		p.advance()
		index = first
		name = p.parseIdent()
	}
	inTok := p.expect(lexer.KwIn, "expected in")
	return &ast.ForStmt{
		For:      start.Pos,
		Index:    index,
		Name:     name,
		In:       inTok.Pos,
		Iterable: p.parseExpr(),
		Body:     p.parseBlock(),
	}
}

func (p *Parser) parseBlock() *ast.Block {
	lbrace := p.expect(lexer.LBrace, "expected {")
	block := &ast.Block{Lbrace: lbrace.Pos}
	for p.cur.Kind != lexer.RBrace && p.cur.Kind != lexer.EOF {
		item := p.parseFormItem()
		if item != nil {
			block.Items = append(block.Items, item)
			continue
		}
		p.error(p.cur.Pos, p.cur.End, "expected block item")
		p.advance()
	}
	rbrace := p.expect(lexer.RBrace, "expected }")
	block.Rbrace = rbrace.End
	return block
}

func (p *Parser) parseBoundValue() (*ast.Ident, ast.TypeExpr, ast.Expr) {
	name := p.parseIdent()
	var typeExpr ast.TypeExpr
	if p.cur.Kind == lexer.Colon {
		p.advance()
		typeExpr = p.parseType()
	}
	p.expect(lexer.Assign, "expected =")
	return name, typeExpr, p.parseExpr()
}

func (p *Parser) parseParams() []*ast.Param {
	p.expect(lexer.LParen, "expected (")
	params := make([]*ast.Param, 0)
	for p.cur.Kind != lexer.RParen && p.cur.Kind != lexer.EOF {
		params = append(params, p.parseParam())
		if p.cur.Kind != lexer.Comma {
			break
		}
		p.advance()
	}
	p.expect(lexer.RParen, "expected )")
	return params
}

func (p *Parser) parseParam() *ast.Param {
	paramName := p.parseIdent()
	p.expect(lexer.Colon, "expected :")
	return &ast.Param{Name: paramName, Type: p.parseType()}
}
