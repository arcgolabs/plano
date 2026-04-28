// Package parser builds plano AST nodes from lexer tokens.
package parser

import (
	"go/token"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/internal/lexer"
)

type Parser struct {
	file  *token.File
	src   []byte
	lex   *lexer.Lexer
	cur   lexer.Token
	peek  lexer.Token
	diags diag.Diagnostics
}

func Parse(file *token.File, src []byte) (*ast.File, diag.Diagnostics) {
	p := &Parser{
		file: file,
		src:  src,
		lex:  lexer.New(file, src),
	}
	p.cur = p.lex.Next()
	p.peek = p.lex.Next()
	node := p.parseFile()
	p.diags.Append(p.lex.Diagnostics())
	return node, p.diags
}

func (p *Parser) parseFile() *ast.File {
	file := &ast.File{
		Start:  p.file.Pos(0),
		Finish: p.file.Pos(len(p.src)),
	}
	for p.cur.Kind != lexer.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			file.Statements = append(file.Statements, stmt)
			continue
		}
		if p.cur.Kind == lexer.EOF {
			break
		}
		p.error(p.cur.Pos, p.cur.End, "unexpected token")
		p.advance()
	}
	return file
}

func (p *Parser) parseStatement() ast.Stmt {
	if stmt := p.parseDeclStatement(); stmt != nil {
		return stmt
	}
	if stmt := p.parseControlStatement(); stmt != nil {
		return stmt
	}
	if p.cur.Kind == lexer.Ident {
		return p.parseFormDeclWithHead(p.parseQualifiedIdent())
	}
	return nil
}

func (p *Parser) parseDeclStatement() ast.Stmt {
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
	return nil
}

func (p *Parser) parseControlStatement() ast.Stmt {
	if p.cur.Kind == lexer.KwIf {
		return p.parseIfStmt()
	}
	if p.cur.Kind == lexer.KwFor {
		return p.parseForStmt()
	}
	return nil
}

func (p *Parser) expect(kind lexer.Kind, message string) lexer.Token {
	if p.cur.Kind == kind {
		tok := p.cur
		p.advance()
		return tok
	}
	p.error(p.cur.Pos, p.cur.End, message)
	tok := p.cur
	if p.cur.Kind != lexer.EOF {
		p.advance()
	}
	return tok
}

func (p *Parser) advance() {
	p.cur = p.peek
	p.peek = p.lex.Next()
}

func (p *Parser) error(pos, end token.Pos, message string) {
	p.diags.AddError(pos, end, message)
}
