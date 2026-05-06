package ast

import "go/token"

type ImportDecl struct {
	Import token.Pos
	Path   *StringLiteral
}

func (n *ImportDecl) Pos() token.Pos { return n.Import }
func (n *ImportDecl) End() token.Pos {
	if n.Path != nil {
		return n.Path.End()
	}
	return n.Import
}
func (*ImportDecl) stmtNode()     {}
func (*ImportDecl) formItemNode() {}

type ConstDecl struct {
	Const token.Pos
	Name  *Ident
	Type  TypeExpr
	Value Expr
}

func (n *ConstDecl) Pos() token.Pos { return n.Const }
func (n *ConstDecl) End() token.Pos {
	if n.Value != nil {
		return n.Value.End()
	}
	if n.Type != nil {
		return n.Type.End()
	}
	if n.Name != nil {
		return n.Name.End()
	}
	return n.Const
}
func (*ConstDecl) stmtNode()     {}
func (*ConstDecl) formItemNode() {}

type LetDecl struct {
	Let   token.Pos
	Name  *Ident
	Type  TypeExpr
	Value Expr
}

func (n *LetDecl) Pos() token.Pos { return n.Let }
func (n *LetDecl) End() token.Pos {
	if n.Value != nil {
		return n.Value.End()
	}
	if n.Type != nil {
		return n.Type.End()
	}
	if n.Name != nil {
		return n.Name.End()
	}
	return n.Let
}
func (*LetDecl) stmtNode()     {}
func (*LetDecl) formItemNode() {}

type FnDecl struct {
	Fn     token.Pos
	Name   *Ident
	Params []*Param
	Result TypeExpr
	Body   *Block
}

func (n *FnDecl) Pos() token.Pos { return n.Fn }
func (n *FnDecl) End() token.Pos {
	if n.Body != nil {
		return n.Body.End()
	}
	if n.Result != nil {
		return n.Result.End()
	}
	if n.Name != nil {
		return n.Name.End()
	}
	return n.Fn
}
func (*FnDecl) stmtNode()     {}
func (*FnDecl) formItemNode() {}

type ReturnStmt struct {
	Return token.Pos
	Value  Expr
}

func (n *ReturnStmt) Pos() token.Pos { return n.Return }
func (n *ReturnStmt) End() token.Pos {
	if n.Value != nil {
		return n.Value.End()
	}
	return n.Return
}
func (*ReturnStmt) stmtNode()     {}
func (*ReturnStmt) formItemNode() {}

type BreakStmt struct {
	Break token.Pos
}

func (n *BreakStmt) Pos() token.Pos { return n.Break }
func (n *BreakStmt) End() token.Pos { return n.Break }
func (*BreakStmt) stmtNode()        {}
func (*BreakStmt) formItemNode()    {}

type ContinueStmt struct {
	Continue token.Pos
}

func (n *ContinueStmt) Pos() token.Pos { return n.Continue }
func (n *ContinueStmt) End() token.Pos { return n.Continue }
func (*ContinueStmt) stmtNode()        {}
func (*ContinueStmt) formItemNode()    {}

type IfStmt struct {
	If        token.Pos
	Condition Expr
	Then      *Block
	Else      *Block
}

func (n *IfStmt) Pos() token.Pos { return n.If }
func (n *IfStmt) End() token.Pos {
	if n.Else != nil {
		return n.Else.End()
	}
	if n.Then != nil {
		return n.Then.End()
	}
	if n.Condition != nil {
		return n.Condition.End()
	}
	return n.If
}
func (*IfStmt) stmtNode()     {}
func (*IfStmt) formItemNode() {}

type ForStmt struct {
	For      token.Pos
	Index    *Ident
	Name     *Ident
	In       token.Pos
	Iterable Expr
	Where    token.Pos
	Filter   Expr
	Body     *Block
}

func (n *ForStmt) Pos() token.Pos { return n.For }
func (n *ForStmt) End() token.Pos {
	if n.Body != nil {
		return n.Body.End()
	}
	if n.Filter != nil {
		return n.Filter.End()
	}
	if n.Iterable != nil {
		return n.Iterable.End()
	}
	if n.Index != nil {
		return n.Index.End()
	}
	if n.Name != nil {
		return n.Name.End()
	}
	return n.For
}
func (*ForStmt) stmtNode()     {}
func (*ForStmt) formItemNode() {}

type FormDecl struct {
	Head  *QualifiedIdent
	Label *Label
	Body  *FormBody
}

func (n *FormDecl) Pos() token.Pos {
	if n.Head != nil {
		return n.Head.Pos()
	}
	return token.NoPos
}
func (n *FormDecl) End() token.Pos {
	if n.Body != nil {
		return n.Body.End()
	}
	if n.Label != nil {
		return n.Label.End()
	}
	if n.Head != nil {
		return n.Head.End()
	}
	return token.NoPos
}
func (*FormDecl) stmtNode()     {}
func (*FormDecl) formItemNode() {}

type Assignment struct {
	Name  *Ident
	Eq    token.Pos
	Value Expr
}

func (n *Assignment) Pos() token.Pos {
	if n.Name != nil {
		return n.Name.Pos()
	}
	return token.NoPos
}
func (n *Assignment) End() token.Pos {
	if n.Value != nil {
		return n.Value.End()
	}
	if n.Name != nil {
		return n.Name.End()
	}
	return token.NoPos
}
func (*Assignment) formItemNode() {}

type CallStmt struct {
	Callee *QualifiedIdent
	Lparen token.Pos
	Rparen token.Pos
	Args   []Expr
}

func (n *CallStmt) Pos() token.Pos {
	if n.Callee != nil {
		return n.Callee.Pos()
	}
	return token.NoPos
}
func (n *CallStmt) End() token.Pos {
	if n.Rparen.IsValid() {
		return n.Rparen
	}
	if n.Callee != nil {
		return n.Callee.End()
	}
	return token.NoPos
}
func (*CallStmt) formItemNode() {}
