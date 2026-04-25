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
