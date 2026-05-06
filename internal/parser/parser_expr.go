package parser

import (
	"strconv"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/internal/lexer"
)

func (p *Parser) parseType() ast.TypeExpr {
	if p.cur.Kind != lexer.Ident {
		p.error(p.cur.Pos, p.cur.End, "expected type")
		return nil
	}
	if typ := p.parseSpecialType(); typ != nil {
		return typ
	}
	name := p.parseQualifiedIdent()
	if len(name.Parts) == 1 {
		return &ast.SimpleType{Name: name.Parts[0]}
	}
	return &ast.QualifiedType{Name: name}
}

func (p *Parser) parseSpecialType() ast.TypeExpr {
	switch {
	case p.cur.Text == "list" && p.peek.Kind == lexer.LT:
		return p.parseListType()
	case p.cur.Text == "map" && p.peek.Kind == lexer.LT:
		return p.parseMapType()
	case p.cur.Text == "ref" && p.peek.Kind == lexer.LT:
		return p.parseRefType()
	default:
		return nil
	}
}

func (p *Parser) parseListType() ast.TypeExpr {
	start := p.cur
	p.advance()
	p.expect(lexer.LT, "expected <")
	elem := p.parseType()
	gt := p.expect(lexer.GT, "expected >")
	return &ast.ListType{List: start.Pos, Elem: elem, Close: gt.End}
}

func (p *Parser) parseMapType() ast.TypeExpr {
	start := p.cur
	p.advance()
	p.expect(lexer.LT, "expected <")
	elem := p.parseType()
	gt := p.expect(lexer.GT, "expected >")
	return &ast.MapType{Map: start.Pos, Elem: elem, Close: gt.End}
}

func (p *Parser) parseRefType() ast.TypeExpr {
	start := p.cur
	p.advance()
	p.expect(lexer.LT, "expected <")
	target := p.parseQualifiedIdent()
	gt := p.expect(lexer.GT, "expected >")
	return &ast.RefType{Ref: start.Pos, Target: target, Close: gt.End}
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseConditionalExpr()
}

func (p *Parser) parseConditionalExpr() ast.Expr {
	condition := p.parseBinaryExpr(1)
	if p.cur.Kind != lexer.Question {
		return condition
	}
	question := p.cur
	p.advance()
	thenExpr := p.parseExpr()
	colon := p.expect(lexer.Colon, "expected :")
	return &ast.ConditionalExpr{
		Condition: condition,
		Question:  question.Pos,
		Then:      thenExpr,
		Colon:     colon.Pos,
		Else:      p.parseExpr(),
	}
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
		left = &ast.BinaryExpr{
			X:     left,
			Op:    op.Text,
			OpPos: op.Pos,
			Y:     p.parseBinaryExpr(prec + 1),
		}
	}
	return left
}

func (p *Parser) parseUnaryExpr() ast.Expr {
	if p.cur.Kind == lexer.Bang || p.cur.Kind == lexer.Minus {
		op := p.cur
		p.advance()
		return &ast.UnaryExpr{
			Op:    op.Text,
			OpPos: op.Pos,
			X:     p.parseUnaryExpr(),
		}
	}
	return p.parsePostfixExpr()
}

func (p *Parser) parsePostfixExpr() ast.Expr {
	expr := p.parsePrimaryExpr()
	for {
		next, ok := p.parsePostfixStep(expr)
		if !ok {
			return expr
		}
		expr = next
	}
}

func (p *Parser) parsePostfixStep(expr ast.Expr) (ast.Expr, bool) {
	if p.cur.Kind == lexer.Dot {
		dot := p.cur
		p.advance()
		return &ast.SelectorExpr{X: expr, Dot: dot.Pos, Sel: p.parseIdent()}, true
	}
	if p.cur.Kind == lexer.LBracket {
		lbrack := p.cur
		p.advance()
		index := p.parseExpr()
		rbrack := p.expect(lexer.RBracket, "expected ]")
		return &ast.IndexExpr{
			X:      expr,
			Lbrack: lbrack.Pos,
			Rbrack: rbrack.End,
			Index:  index,
		}, true
	}
	if p.cur.Kind == lexer.LParen {
		return p.parseCallExpr(expr), true
	}
	return nil, false
}

func (p *Parser) parseCallExpr(fun ast.Expr) ast.Expr {
	lparen := p.expect(lexer.LParen, "expected (")
	args := p.parseExprList(lexer.RParen)
	rparen := p.expect(lexer.RParen, "expected )")
	return &ast.CallExpr{
		Fun:    fun,
		Lparen: lparen.Pos,
		Rparen: rparen.End,
		Args:   args,
	}
}

func (p *Parser) parsePrimaryExpr() ast.Expr {
	if expr := p.parseScalarLiteral(); expr != nil {
		return expr
	}
	if p.cur.Kind == lexer.Ident {
		return &ast.IdentExpr{Name: p.parseIdent()}
	}
	if p.cur.Kind == lexer.LBracket {
		return p.parseArrayExpr()
	}
	if p.cur.Kind == lexer.LBrace {
		return p.parseObjectExpr()
	}
	if p.cur.Kind == lexer.LParen {
		lparen := p.cur
		p.advance()
		x := p.parseExpr()
		rparen := p.expect(lexer.RParen, "expected )")
		return &ast.ParenExpr{
			Lparen: lparen.Pos,
			Rparen: rparen.End,
			X:      x,
		}
	}
	return p.parseBadExpr()
}

func (p *Parser) parseScalarLiteral() ast.Expr {
	if p.cur.Kind == lexer.String {
		return p.parseStringLiteral()
	}
	if p.cur.Kind == lexer.Int {
		return p.parseIntLiteral()
	}
	if p.cur.Kind == lexer.Float {
		return p.parseFloatLiteral()
	}
	if p.cur.Kind == lexer.Duration {
		return p.parseRawLiteral(func(tok lexer.Token) ast.Expr {
			return &ast.DurationLiteral{Start: tok.Pos, Finish: tok.End, Raw: tok.Text}
		})
	}
	if p.cur.Kind == lexer.Size {
		return p.parseRawLiteral(func(tok lexer.Token) ast.Expr {
			return &ast.SizeLiteral{Start: tok.Pos, Finish: tok.End, Raw: tok.Text}
		})
	}
	if p.cur.Kind == lexer.KwTrue {
		return p.parseBoolLiteral(true)
	}
	if p.cur.Kind == lexer.KwFalse {
		return p.parseBoolLiteral(false)
	}
	if p.cur.Kind == lexer.KwNull {
		tok := p.cur
		p.advance()
		return &ast.NullLiteral{Start: tok.Pos, Finish: tok.End}
	}
	return nil
}

func (p *Parser) parseIntLiteral() ast.Expr {
	tok := p.cur
	p.advance()
	value, err := strconv.ParseInt(tok.Text, 10, 64)
	if err != nil {
		p.error(tok.Pos, tok.End, "invalid integer literal")
	}
	return &ast.IntLiteral{Start: tok.Pos, Finish: tok.End, Raw: tok.Text, Value: value}
}

func (p *Parser) parseFloatLiteral() ast.Expr {
	tok := p.cur
	p.advance()
	value, err := strconv.ParseFloat(tok.Text, 64)
	if err != nil {
		p.error(tok.Pos, tok.End, "invalid float literal")
	}
	return &ast.FloatLiteral{Start: tok.Pos, Finish: tok.End, Raw: tok.Text, Value: value}
}

func (p *Parser) parseRawLiteral(build func(tok lexer.Token) ast.Expr) ast.Expr {
	tok := p.cur
	p.advance()
	return build(tok)
}

func (p *Parser) parseBoolLiteral(value bool) ast.Expr {
	tok := p.cur
	p.advance()
	return &ast.BoolLiteral{Start: tok.Pos, Finish: tok.End, Value: value}
}

func (p *Parser) parseBadExpr() ast.Expr {
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
