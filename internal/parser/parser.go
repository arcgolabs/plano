// Package parser builds plano AST nodes from lexer tokens.
//
//nolint:cyclop,gocognit,gocyclo,funlen,exhaustive,revive // The hand-written parser intentionally keeps grammar dispatch in one file.
package parser

import (
	"go/token"
	"strconv"

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
	switch p.cur.Kind {
	case lexer.KwImport:
		return p.parseImportDecl()
	case lexer.KwConst:
		return p.parseConstDecl()
	case lexer.KwLet:
		return p.parseLetDecl()
	case lexer.KwFn:
		return p.parseFnDecl()
	case lexer.KwReturn:
		return p.parseReturnStmt()
	case lexer.KwIf:
		return p.parseIfStmt()
	case lexer.KwFor:
		return p.parseForStmt()
	case lexer.Ident:
		head := p.parseQualifiedIdent()
		return p.parseFormDeclWithHead(head)
	default:
		return nil
	}
}

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
	name := p.parseIdent()
	var typeExpr ast.TypeExpr
	if p.cur.Kind == lexer.Colon {
		p.advance()
		typeExpr = p.parseType()
	}
	p.expect(lexer.Assign, "expected =")
	value := p.parseExpr()
	return &ast.ConstDecl{
		Const: start.Pos,
		Name:  name,
		Type:  typeExpr,
		Value: value,
	}
}

func (p *Parser) parseLetDecl() *ast.LetDecl {
	start := p.expect(lexer.KwLet, "expected let")
	name := p.parseIdent()
	var typeExpr ast.TypeExpr
	if p.cur.Kind == lexer.Colon {
		p.advance()
		typeExpr = p.parseType()
	}
	p.expect(lexer.Assign, "expected =")
	value := p.parseExpr()
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
	p.expect(lexer.LParen, "expected (")
	var params []*ast.Param
	for p.cur.Kind != lexer.RParen && p.cur.Kind != lexer.EOF {
		paramName := p.parseIdent()
		p.expect(lexer.Colon, "expected :")
		paramType := p.parseType()
		params = append(params, &ast.Param{Name: paramName, Type: paramType})
		if p.cur.Kind != lexer.Comma {
			break
		}
		p.advance()
	}
	p.expect(lexer.RParen, "expected )")
	var result ast.TypeExpr
	if p.cur.Kind == lexer.Colon {
		p.advance()
		result = p.parseType()
	}
	body := p.parseBlock()
	return &ast.FnDecl{
		Fn:     start.Pos,
		Name:   name,
		Params: params,
		Result: result,
		Body:   body,
	}
}

func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	start := p.expect(lexer.KwReturn, "expected return")
	return &ast.ReturnStmt{
		Return: start.Pos,
		Value:  p.parseExpr(),
	}
}

func (p *Parser) parseIfStmt() *ast.IfStmt {
	start := p.expect(lexer.KwIf, "expected if")
	condition := p.parseExpr()
	thenBlock := p.parseBlock()
	var elseBlock *ast.Block
	if p.cur.Kind == lexer.KwElse {
		p.advance()
		elseBlock = p.parseBlock()
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
	name := p.parseIdent()
	inTok := p.expect(lexer.KwIn, "expected in")
	iterable := p.parseExpr()
	body := p.parseBlock()
	return &ast.ForStmt{
		For:      start.Pos,
		Name:     name,
		In:       inTok.Pos,
		Iterable: iterable,
		Body:     body,
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

func (p *Parser) parseFormDeclWithHead(head *ast.QualifiedIdent) *ast.FormDecl {
	var label *ast.Label
	switch p.cur.Kind {
	case lexer.Ident:
		id := p.parseIdent()
		label = &ast.Label{
			Start:  id.Pos(),
			Finish: id.End(),
			Value:  id.Name,
		}
	case lexer.String:
		tok := p.cur
		p.advance()
		label = &ast.Label{
			Start:  tok.Pos,
			Finish: tok.End,
			Value:  tok.Text,
			Quoted: true,
		}
	}
	body := p.parseFormBody()
	return &ast.FormDecl{
		Head:  head,
		Label: label,
		Body:  body,
	}
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
	switch p.cur.Kind {
	case lexer.KwImport:
		return p.parseImportDecl()
	case lexer.KwConst:
		return p.parseConstDecl()
	case lexer.KwLet:
		return p.parseLetDecl()
	case lexer.KwFn:
		return p.parseFnDecl()
	case lexer.KwReturn:
		return p.parseReturnStmt()
	case lexer.KwIf:
		return p.parseIfStmt()
	case lexer.KwFor:
		return p.parseForStmt()
	case lexer.Ident:
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
	default:
		return nil
	}
}

func (p *Parser) parseCallStmtWithHead(head *ast.QualifiedIdent) ast.FormItem {
	lparen := p.expect(lexer.LParen, "expected (")
	var args []ast.Expr
	for p.cur.Kind != lexer.RParen && p.cur.Kind != lexer.EOF {
		args = append(args, p.parseExpr())
		if p.cur.Kind != lexer.Comma {
			break
		}
		p.advance()
	}
	rparen := p.expect(lexer.RParen, "expected )")
	return &ast.CallStmt{
		Callee: head,
		Lparen: lparen.Pos,
		Rparen: rparen.End,
		Args:   args,
	}
}

func (p *Parser) parseType() ast.TypeExpr {
	if p.cur.Kind != lexer.Ident {
		p.error(p.cur.Pos, p.cur.End, "expected type")
		return nil
	}
	if p.cur.Text == "list" && p.peek.Kind == lexer.LT {
		start := p.cur
		p.advance()
		p.expect(lexer.LT, "expected <")
		elem := p.parseType()
		gt := p.expect(lexer.GT, "expected >")
		return &ast.ListType{List: start.Pos, Elem: elem, Close: gt.End}
	}
	if p.cur.Text == "map" && p.peek.Kind == lexer.LT {
		start := p.cur
		p.advance()
		p.expect(lexer.LT, "expected <")
		elem := p.parseType()
		gt := p.expect(lexer.GT, "expected >")
		return &ast.MapType{Map: start.Pos, Elem: elem, Close: gt.End}
	}
	if p.cur.Text == "ref" && p.peek.Kind == lexer.LT {
		start := p.cur
		p.advance()
		p.expect(lexer.LT, "expected <")
		target := p.parseQualifiedIdent()
		gt := p.expect(lexer.GT, "expected >")
		return &ast.RefType{Ref: start.Pos, Target: target, Close: gt.End}
	}

	name := p.parseQualifiedIdent()
	if len(name.Parts) == 1 {
		return &ast.SimpleType{Name: name.Parts[0]}
	}
	return &ast.QualifiedType{Name: name}
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseBinaryExpr(1)
}

func (p *Parser) parseBinaryExpr(minPrec int) ast.Expr {
	left := p.parseUnaryExpr()
	for {
		prec := precedence(p.cur.Kind)
		if prec < minPrec {
			break
		}
		op := p.cur
		p.advance()
		right := p.parseBinaryExpr(prec + 1)
		left = &ast.BinaryExpr{
			X:     left,
			Op:    op.Text,
			OpPos: op.Pos,
			Y:     right,
		}
	}
	return left
}

func (p *Parser) parseUnaryExpr() ast.Expr {
	switch p.cur.Kind {
	case lexer.Bang, lexer.Minus:
		op := p.cur
		p.advance()
		return &ast.UnaryExpr{
			Op:    op.Text,
			OpPos: op.Pos,
			X:     p.parseUnaryExpr(),
		}
	default:
		return p.parsePostfixExpr()
	}
}

func (p *Parser) parsePostfixExpr() ast.Expr {
	expr := p.parsePrimaryExpr()
	for {
		switch p.cur.Kind {
		case lexer.Dot:
			dot := p.cur
			p.advance()
			sel := p.parseIdent()
			expr = &ast.SelectorExpr{X: expr, Dot: dot.Pos, Sel: sel}
		case lexer.LBracket:
			lbrack := p.cur
			p.advance()
			index := p.parseExpr()
			rbrack := p.expect(lexer.RBracket, "expected ]")
			expr = &ast.IndexExpr{
				X:      expr,
				Lbrack: lbrack.Pos,
				Rbrack: rbrack.End,
				Index:  index,
			}
		case lexer.LParen:
			lparen := p.cur
			p.advance()
			var args []ast.Expr
			for p.cur.Kind != lexer.RParen && p.cur.Kind != lexer.EOF {
				args = append(args, p.parseExpr())
				if p.cur.Kind != lexer.Comma {
					break
				}
				p.advance()
			}
			rparen := p.expect(lexer.RParen, "expected )")
			expr = &ast.CallExpr{
				Fun:    expr,
				Lparen: lparen.Pos,
				Rparen: rparen.End,
				Args:   args,
			}
		default:
			return expr
		}
	}
}

func (p *Parser) parsePrimaryExpr() ast.Expr {
	switch p.cur.Kind {
	case lexer.String:
		return p.parseStringLiteral()
	case lexer.Int:
		tok := p.cur
		p.advance()
		value, err := strconv.ParseInt(tok.Text, 10, 64)
		if err != nil {
			p.error(tok.Pos, tok.End, "invalid integer literal")
		}
		return &ast.IntLiteral{Start: tok.Pos, Finish: tok.End, Raw: tok.Text, Value: value}
	case lexer.Float:
		tok := p.cur
		p.advance()
		value, err := strconv.ParseFloat(tok.Text, 64)
		if err != nil {
			p.error(tok.Pos, tok.End, "invalid float literal")
		}
		return &ast.FloatLiteral{Start: tok.Pos, Finish: tok.End, Raw: tok.Text, Value: value}
	case lexer.Duration:
		tok := p.cur
		p.advance()
		return &ast.DurationLiteral{Start: tok.Pos, Finish: tok.End, Raw: tok.Text}
	case lexer.Size:
		tok := p.cur
		p.advance()
		return &ast.SizeLiteral{Start: tok.Pos, Finish: tok.End, Raw: tok.Text}
	case lexer.KwTrue:
		tok := p.cur
		p.advance()
		return &ast.BoolLiteral{Start: tok.Pos, Finish: tok.End, Value: true}
	case lexer.KwFalse:
		tok := p.cur
		p.advance()
		return &ast.BoolLiteral{Start: tok.Pos, Finish: tok.End, Value: false}
	case lexer.KwNull:
		tok := p.cur
		p.advance()
		return &ast.NullLiteral{Start: tok.Pos, Finish: tok.End}
	case lexer.Ident:
		return &ast.IdentExpr{Name: p.parseIdent()}
	case lexer.LBracket:
		return p.parseArrayExpr()
	case lexer.LBrace:
		return p.parseObjectExpr()
	case lexer.LParen:
		lparen := p.cur
		p.advance()
		x := p.parseExpr()
		rparen := p.expect(lexer.RParen, "expected )")
		return &ast.ParenExpr{
			Lparen: lparen.Pos,
			Rparen: rparen.End,
			X:      x,
		}
	default:
		p.error(p.cur.Pos, p.cur.End, "expected expression")
		bad := p.cur
		p.advance()
		return &ast.IdentExpr{
			Name: &ast.Ident{
				Name:   bad.Text,
				Start:  bad.Pos,
				Finish: bad.End,
			},
		}
	}
}

func (p *Parser) parseArrayExpr() ast.Expr {
	lbrack := p.expect(lexer.LBracket, "expected [")
	array := &ast.ArrayExpr{Lbrack: lbrack.Pos}
	for p.cur.Kind != lexer.RBracket && p.cur.Kind != lexer.EOF {
		array.Elements = append(array.Elements, p.parseExpr())
		if p.cur.Kind != lexer.Comma {
			break
		}
		p.advance()
		if p.cur.Kind == lexer.RBracket {
			break
		}
	}
	rbrack := p.expect(lexer.RBracket, "expected ]")
	array.Rbrack = rbrack.End
	return array
}

func (p *Parser) parseObjectExpr() ast.Expr {
	lbrace := p.expect(lexer.LBrace, "expected {")
	object := &ast.ObjectExpr{Lbrace: lbrace.Pos}
	for p.cur.Kind != lexer.RBrace && p.cur.Kind != lexer.EOF {
		key := p.parseIdent()
		p.expect(lexer.Assign, "expected =")
		object.Entries = append(object.Entries, &ast.ObjectEntry{
			Key:   key,
			Value: p.parseExpr(),
		})
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

func precedence(kind lexer.Kind) int {
	switch kind {
	case lexer.OrOr:
		return 1
	case lexer.AndAnd:
		return 2
	case lexer.Eq, lexer.NotEq:
		return 3
	case lexer.GT, lexer.GTE, lexer.LT, lexer.LTE:
		return 4
	case lexer.Plus, lexer.Minus:
		return 5
	case lexer.Star, lexer.Slash, lexer.Percent:
		return 6
	default:
		return 0
	}
}
