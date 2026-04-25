package ast

import "go/token"

type StringLiteral struct {
	Start  token.Pos
	Finish token.Pos
	Value  string
}

func (n *StringLiteral) Pos() token.Pos { return n.Start }
func (n *StringLiteral) End() token.Pos { return n.Finish }
func (*StringLiteral) exprNode()        {}

type IntLiteral struct {
	Start  token.Pos
	Finish token.Pos
	Raw    string
	Value  int64
}

func (n *IntLiteral) Pos() token.Pos { return n.Start }
func (n *IntLiteral) End() token.Pos { return n.Finish }
func (*IntLiteral) exprNode()        {}

type FloatLiteral struct {
	Start  token.Pos
	Finish token.Pos
	Raw    string
	Value  float64
}

func (n *FloatLiteral) Pos() token.Pos { return n.Start }
func (n *FloatLiteral) End() token.Pos { return n.Finish }
func (*FloatLiteral) exprNode()        {}

type BoolLiteral struct {
	Start  token.Pos
	Finish token.Pos
	Value  bool
}

func (n *BoolLiteral) Pos() token.Pos { return n.Start }
func (n *BoolLiteral) End() token.Pos { return n.Finish }
func (*BoolLiteral) exprNode()        {}

type NullLiteral struct {
	Start  token.Pos
	Finish token.Pos
}

func (n *NullLiteral) Pos() token.Pos { return n.Start }
func (n *NullLiteral) End() token.Pos { return n.Finish }
func (*NullLiteral) exprNode()        {}

type DurationLiteral struct {
	Start  token.Pos
	Finish token.Pos
	Raw    string
}

func (n *DurationLiteral) Pos() token.Pos { return n.Start }
func (n *DurationLiteral) End() token.Pos { return n.Finish }
func (*DurationLiteral) exprNode()        {}

type SizeLiteral struct {
	Start  token.Pos
	Finish token.Pos
	Raw    string
}

func (n *SizeLiteral) Pos() token.Pos { return n.Start }
func (n *SizeLiteral) End() token.Pos { return n.Finish }
func (*SizeLiteral) exprNode()        {}

type IdentExpr struct {
	Name *Ident
}

func (n *IdentExpr) Pos() token.Pos {
	if n == nil || n.Name == nil {
		return token.NoPos
	}
	return n.Name.Pos()
}

func (n *IdentExpr) End() token.Pos {
	if n == nil || n.Name == nil {
		return token.NoPos
	}
	return n.Name.End()
}

func (*IdentExpr) exprNode() {}
