package ast

import (
	"go/token"
	"strings"
)

type Node interface {
	Pos() token.Pos
	End() token.Pos
}

type Stmt interface {
	Node
	stmtNode()
}

type FormItem interface {
	Node
	formItemNode()
}

type Expr interface {
	Node
	exprNode()
}

type TypeExpr interface {
	Node
	typeExprNode()
}

type File struct {
	Start      token.Pos
	Finish     token.Pos
	Statements []Stmt
}

func (n *File) Pos() token.Pos { return n.Start }
func (n *File) End() token.Pos { return n.Finish }

type Ident struct {
	Name   string
	Start  token.Pos
	Finish token.Pos
}

func (n *Ident) Pos() token.Pos { return n.Start }
func (n *Ident) End() token.Pos { return n.Finish }

type QualifiedIdent struct {
	Parts []*Ident
}

func (n *QualifiedIdent) Pos() token.Pos {
	if n == nil || len(n.Parts) == 0 {
		return token.NoPos
	}
	return n.Parts[0].Pos()
}

func (n *QualifiedIdent) End() token.Pos {
	if n == nil || len(n.Parts) == 0 {
		return token.NoPos
	}
	return n.Parts[len(n.Parts)-1].End()
}

func (n *QualifiedIdent) String() string {
	if n == nil {
		return ""
	}
	names := make([]string, 0, len(n.Parts))
	for _, part := range n.Parts {
		names = append(names, part.Name)
	}
	return strings.Join(names, ".")
}

type Label struct {
	Start  token.Pos
	Finish token.Pos
	Value  string
	Quoted bool
}

func (n *Label) Pos() token.Pos { return n.Start }
func (n *Label) End() token.Pos { return n.Finish }

type Param struct {
	Name *Ident
	Type TypeExpr
}

func (n *Param) Pos() token.Pos {
	if n == nil || n.Name == nil {
		return token.NoPos
	}
	return n.Name.Pos()
}

func (n *Param) End() token.Pos {
	if n == nil {
		return token.NoPos
	}
	if n.Type != nil {
		return n.Type.End()
	}
	if n.Name != nil {
		return n.Name.End()
	}
	return token.NoPos
}

type Block struct {
	Lbrace token.Pos
	Rbrace token.Pos
	Items  []FormItem
}

func (n *Block) Pos() token.Pos { return n.Lbrace }
func (n *Block) End() token.Pos { return n.Rbrace }

type FormBody struct {
	Lbrace token.Pos
	Rbrace token.Pos
	Items  []FormItem
}

func (n *FormBody) Pos() token.Pos { return n.Lbrace }
func (n *FormBody) End() token.Pos { return n.Rbrace }

type ObjectEntry struct {
	Key   *Ident
	Value Expr
}

func (n *ObjectEntry) Pos() token.Pos {
	if n == nil || n.Key == nil {
		return token.NoPos
	}
	return n.Key.Pos()
}

func (n *ObjectEntry) End() token.Pos {
	if n == nil {
		return token.NoPos
	}
	if n.Value != nil {
		return n.Value.End()
	}
	if n.Key != nil {
		return n.Key.End()
	}
	return token.NoPos
}
