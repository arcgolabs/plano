package parser

import (
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/internal/lexer"
)

func (p *Parser) parseFormDeclWithHead(head *ast.QualifiedIdent) *ast.FormDecl {
	label := p.parseOptionalLabel()
	return &ast.FormDecl{
		Head:  head,
		Label: label,
		Body:  p.parseFormBody(),
	}
}

func (p *Parser) parseOptionalLabel() *ast.Label {
	if p.cur.Kind == lexer.Ident {
		id := p.parseIdent()
		return &ast.Label{
			Start:  id.Pos(),
			Finish: id.End(),
			Value:  id.Name,
		}
	}
	if p.cur.Kind == lexer.String {
		tok := p.cur
		p.advance()
		return &ast.Label{
			Start:  tok.Pos,
			Finish: tok.End,
			Value:  tok.Text,
			Quoted: true,
		}
	}
	return nil
}

func (p *Parser) parseFormBody() *ast.FormBody {
	lbrace := p.expect(lexer.LBrace, "expected {")
	body := &ast.FormBody{Lbrace: lbrace.Pos}
	for p.cur.Kind != lexer.RBrace && p.cur.Kind != lexer.EOF {
		item := p.parseFormItem()
		if item != nil {
			body.Items = append(body.Items, item)
			continue
		}
		p.error(p.cur.Pos, p.cur.End, "expected form item")
		p.advance()
	}
	rbrace := p.expect(lexer.RBrace, "expected }")
	body.Rbrace = rbrace.End
	return body
}

func (p *Parser) parseFormItem() ast.FormItem {
	if item := p.parseKeywordFormItem(); item != nil {
		return item
	}
	if p.cur.Kind == lexer.Ident {
		return p.parseIdentFormItem()
	}
	return nil
}

func (p *Parser) parseKeywordFormItem() ast.FormItem {
	if p.cur.Kind == lexer.KwImport {
		return p.parseImportDecl()
	}
	if p.cur.Kind == lexer.KwConst {
		return p.parseConstDecl()
	}
	if p.cur.Kind == lexer.KwLet {
		return p.parseLetDecl()
	}
	if p.cur.Kind == lexer.KwFn {
		return p.parseFnDecl()
	}
	if p.cur.Kind == lexer.KwReturn {
		return p.parseReturnStmt()
	}
	if p.cur.Kind == lexer.KwBreak {
		return p.parseBreakStmt()
	}
	if p.cur.Kind == lexer.KwContinue {
		return p.parseContinueStmt()
	}
	if p.cur.Kind == lexer.KwIf {
		return p.parseIfStmt()
	}
	if p.cur.Kind == lexer.KwFor {
		return p.parseForStmt()
	}
	return nil
}

func (p *Parser) parseIdentFormItem() ast.FormItem {
	head := p.parseQualifiedIdent()
	if len(head.Parts) == 1 && p.cur.Kind == lexer.Assign {
		eq := p.expect(lexer.Assign, "expected =")
		return &ast.Assignment{
			Name:  head.Parts[0],
			Eq:    eq.Pos,
			Value: p.parseExpr(),
		}
	}
	if p.cur.Kind == lexer.LParen {
		return p.parseCallStmtWithHead(head)
	}
	return p.parseFormDeclWithHead(head)
}

func (p *Parser) parseCallStmtWithHead(head *ast.QualifiedIdent) ast.FormItem {
	lparen := p.expect(lexer.LParen, "expected (")
	args := p.parseExprList(lexer.RParen)
	rparen := p.expect(lexer.RParen, "expected )")
	return &ast.CallStmt{
		Callee: head,
		Lparen: lparen.Pos,
		Rparen: rparen.End,
		Args:   args,
	}
}
