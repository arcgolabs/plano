package compiler

import (
	"go/token"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

type Binding struct {
	Files     list.List[string]
	Scopes    *mapping.OrderedMap[string, ScopeBinding]
	Locals    *mapping.OrderedMap[string, LocalBinding]
	Uses      *mapping.OrderedMap[string, NameUse]
	Symbols   *mapping.OrderedMap[string, Symbol]
	Consts    *mapping.OrderedMap[string, ConstBinding]
	Functions *mapping.OrderedMap[string, FunctionBinding]
}

type ScopeKind string

const (
	ScopeModule   ScopeKind = "module"
	ScopeFile     ScopeKind = "file"
	ScopeForm     ScopeKind = "form"
	ScopeFunction ScopeKind = "function"
	ScopeBlock    ScopeKind = "block"
	ScopeLoop     ScopeKind = "loop"
)

type ScopeBinding struct {
	ID       string    `json:"id"                 yaml:"id"`
	Kind     ScopeKind `json:"kind"               yaml:"kind"`
	ParentID string    `json:"parentId,omitempty" yaml:"parentId,omitempty"`
	Pos      token.Pos `json:"pos"                yaml:"pos"`
	End      token.Pos `json:"end"                yaml:"end"`
}

type ConstBinding struct {
	Name string
	Type schema.Type
	Pos  token.Pos
	End  token.Pos
}

type LocalBindingKind string

const (
	LocalConst LocalBindingKind = "const"
	LocalLet   LocalBindingKind = "let"
	LocalParam LocalBindingKind = "param"
	LocalLoop  LocalBindingKind = "loop"
)

type LocalBinding struct {
	ID      string           `json:"id"      yaml:"id"`
	Name    string           `json:"name"    yaml:"name"`
	Kind    LocalBindingKind `json:"kind"    yaml:"kind"`
	ScopeID string           `json:"scopeId" yaml:"scopeId"`
	Type    schema.Type      `json:"type"    yaml:"type"`
	Pos     token.Pos        `json:"pos"     yaml:"pos"`
	End     token.Pos        `json:"end"     yaml:"end"`
}

type FunctionBinding struct {
	Name   string
	Params list.List[ParamBinding]
	Result schema.Type
	Pos    token.Pos
	End    token.Pos
}

type ParamBinding struct {
	Name string
	Type schema.Type
	Pos  token.Pos
	End  token.Pos
}

type NameUseKind string

const (
	UseLocal           NameUseKind = "local"
	UseConst           NameUseKind = "const"
	UseFunction        NameUseKind = "function"
	UseBuiltinFunction NameUseKind = "builtin-function"
	UseSymbol          NameUseKind = "symbol"
	UseGlobal          NameUseKind = "global"
	UseAction          NameUseKind = "action"
	UseUnresolved      NameUseKind = "unresolved"
)

type NameUse struct {
	ID       string      `json:"id"                 yaml:"id"`
	Name     string      `json:"name"               yaml:"name"`
	Kind     NameUseKind `json:"kind"               yaml:"kind"`
	ScopeID  string      `json:"scopeId,omitempty"  yaml:"scopeId,omitempty"`
	TargetID string      `json:"targetId,omitempty" yaml:"targetId,omitempty"`
	Pos      token.Pos   `json:"pos"                yaml:"pos"`
	End      token.Pos   `json:"end"                yaml:"end"`
}

type BindResult struct {
	Binding     *Binding
	FileSet     *token.FileSet
	Diagnostics diag.Diagnostics
}

type preparedInput struct {
	fileSet     *token.FileSet
	units       []parsedUnit
	diagnostics diag.Diagnostics
}

type boundIndex struct {
	binding     *Binding
	symbols     *mapping.OrderedMap[string, Symbol]
	constDecls  *mapping.OrderedMap[string, *ast.ConstDecl]
	funcDecls   *mapping.OrderedMap[string, *ast.FnDecl]
	diagnostics diag.Diagnostics
}

type binder struct {
	compiler   *Compiler
	binding    *Binding
	symbols    *mapping.OrderedMap[string, Symbol]
	constDecls *mapping.OrderedMap[string, *ast.ConstDecl]
	funcDecls  *mapping.OrderedMap[string, *ast.FnDecl]
	nextScope  int
	nextLocal  int
	nextUse    int
	diags      diag.Diagnostics
}
