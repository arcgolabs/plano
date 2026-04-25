// Package ast defines the syntax tree nodes produced by the plano parser.
//
//nolint:revive // The expression node definitions are intentionally grouped in one file.
package ast

import "go/token"

type SimpleType struct {
	Name *Ident
}

func (n *SimpleType) Pos() token.Pos {
	if n == nil || n.Name == nil {
		return token.NoPos
	}
	return n.Name.Pos()
}
func (n *SimpleType) End() token.Pos {
	if n == nil || n.Name == nil {
		return token.NoPos
	}
	return n.Name.End()
}
func (*SimpleType) typeExprNode() {}

type QualifiedType struct {
	Name *QualifiedIdent
}

func (n *QualifiedType) Pos() token.Pos {
	if n == nil || n.Name == nil {
		return token.NoPos
	}
	return n.Name.Pos()
}
func (n *QualifiedType) End() token.Pos {
	if n == nil || n.Name == nil {
		return token.NoPos
	}
	return n.Name.End()
}
func (*QualifiedType) typeExprNode() {}

type ListType struct {
	List  token.Pos
	Elem  TypeExpr
	Close token.Pos
}

func (n *ListType) Pos() token.Pos { return n.List }
func (n *ListType) End() token.Pos {
	if n.Close.IsValid() {
		return n.Close
	}
	if n.Elem != nil {
		return n.Elem.End()
	}
	return n.List
}
func (*ListType) typeExprNode() {}

type MapType struct {
	Map   token.Pos
	Elem  TypeExpr
	Close token.Pos
}

func (n *MapType) Pos() token.Pos { return n.Map }
func (n *MapType) End() token.Pos {
	if n.Close.IsValid() {
		return n.Close
	}
	if n.Elem != nil {
		return n.Elem.End()
	}
	return n.Map
}
func (*MapType) typeExprNode() {}

type RefType struct {
	Ref    token.Pos
	Target *QualifiedIdent
	Close  token.Pos
}

func (n *RefType) Pos() token.Pos { return n.Ref }
func (n *RefType) End() token.Pos {
	if n.Close.IsValid() {
		return n.Close
	}
	if n.Target != nil {
		return n.Target.End()
	}
	return n.Ref
}
func (*RefType) typeExprNode() {}

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
