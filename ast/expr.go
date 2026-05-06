// Package ast defines the syntax tree nodes produced by the plano parser.
package ast

import "go/token"

type ArrayExpr struct {
	Lbrack   token.Pos
	Rbrack   token.Pos
	Elements []Expr
}

func (n *ArrayExpr) Pos() token.Pos { return n.Lbrack }
func (n *ArrayExpr) End() token.Pos { return n.Rbrack }
func (*ArrayExpr) exprNode()        {}

type ObjectExpr struct {
	Lbrace  token.Pos
	Rbrace  token.Pos
	Entries []*ObjectEntry
}

func (n *ObjectExpr) Pos() token.Pos { return n.Lbrace }
func (n *ObjectExpr) End() token.Pos { return n.Rbrace }
func (*ObjectExpr) exprNode()        {}

type ParenExpr struct {
	Lparen token.Pos
	Rparen token.Pos
	X      Expr
}

func (n *ParenExpr) Pos() token.Pos { return n.Lparen }
func (n *ParenExpr) End() token.Pos { return n.Rparen }
func (*ParenExpr) exprNode()        {}

type UnaryExpr struct {
	Op    string
	OpPos token.Pos
	X     Expr
}

func (n *UnaryExpr) Pos() token.Pos { return n.OpPos }
func (n *UnaryExpr) End() token.Pos {
	if n.X != nil {
		return n.X.End()
	}
	return n.OpPos
}
func (*UnaryExpr) exprNode() {}

type BinaryExpr struct {
	X     Expr
	Op    string
	OpPos token.Pos
	Y     Expr
}

func (n *BinaryExpr) Pos() token.Pos {
	if n.X != nil {
		return n.X.Pos()
	}
	return n.OpPos
}
func (n *BinaryExpr) End() token.Pos {
	if n.Y != nil {
		return n.Y.End()
	}
	if n.X != nil {
		return n.X.End()
	}
	return n.OpPos
}
func (*BinaryExpr) exprNode() {}

type ConditionalExpr struct {
	Condition Expr
	Question  token.Pos
	Then      Expr
	Colon     token.Pos
	Else      Expr
}

func (n *ConditionalExpr) Pos() token.Pos {
	if n.Condition != nil {
		return n.Condition.Pos()
	}
	return n.Question
}
func (n *ConditionalExpr) End() token.Pos {
	if n.Else != nil {
		return n.Else.End()
	}
	if n.Then != nil {
		return n.Then.End()
	}
	if n.Condition != nil {
		return n.Condition.End()
	}
	return n.Question
}
func (*ConditionalExpr) exprNode() {}

type SelectorExpr struct {
	X   Expr
	Dot token.Pos
	Sel *Ident
}

func (n *SelectorExpr) Pos() token.Pos {
	if n.X != nil {
		return n.X.Pos()
	}
	return n.Dot
}
func (n *SelectorExpr) End() token.Pos {
	if n.Sel != nil {
		return n.Sel.End()
	}
	if n.X != nil {
		return n.X.End()
	}
	return n.Dot
}
func (*SelectorExpr) exprNode() {}

type IndexExpr struct {
	X      Expr
	Lbrack token.Pos
	Rbrack token.Pos
	Index  Expr
}

func (n *IndexExpr) Pos() token.Pos {
	if n.X != nil {
		return n.X.Pos()
	}
	return n.Lbrack
}
func (n *IndexExpr) End() token.Pos { return n.Rbrack }
func (*IndexExpr) exprNode()        {}

type CallExpr struct {
	Fun    Expr
	Lparen token.Pos
	Rparen token.Pos
	Args   []Expr
}

func (n *CallExpr) Pos() token.Pos {
	if n.Fun != nil {
		return n.Fun.Pos()
	}
	return n.Lparen
}
func (n *CallExpr) End() token.Pos { return n.Rparen }
func (*CallExpr) exprNode()        {}
