package compiler

import (
	"go/token"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

type CheckInfo struct {
	Exprs  *mapping.OrderedMap[string, ExprCheck]
	Fields *mapping.OrderedMap[string, FieldCheck]
	Calls  *mapping.OrderedMap[string, CallCheck]
}

type ExprCheck struct {
	ID      string      `json:"id"      yaml:"id"`
	Kind    string      `json:"kind"    yaml:"kind"`
	ScopeID string      `json:"scopeId" yaml:"scopeId"`
	Type    schema.Type `json:"type"    yaml:"type"`
	Pos     token.Pos   `json:"pos"     yaml:"pos"`
	End     token.Pos   `json:"end"     yaml:"end"`
}

type FieldCheck struct {
	ID       string      `json:"id"       yaml:"id"`
	FormKind string      `json:"formKind" yaml:"formKind"`
	Field    string      `json:"field"    yaml:"field"`
	ScopeID  string      `json:"scopeId"  yaml:"scopeId"`
	Expected schema.Type `json:"expected" yaml:"expected"`
	Actual   schema.Type `json:"actual"   yaml:"actual"`
	Pos      token.Pos   `json:"pos"      yaml:"pos"`
	End      token.Pos   `json:"end"      yaml:"end"`
}

type CallCheck struct {
	ID      string                 `json:"id"      yaml:"id"`
	Name    string                 `json:"name"    yaml:"name"`
	ScopeID string                 `json:"scopeId" yaml:"scopeId"`
	Args    list.List[schema.Type] `json:"args"    yaml:"args"`
	Result  schema.Type            `json:"result"  yaml:"result"`
	Pos     token.Pos              `json:"pos"     yaml:"pos"`
	End     token.Pos              `json:"end"     yaml:"end"`
}

type CheckResult struct {
	Binding     *Binding
	Checks      *CheckInfo
	FileSet     *token.FileSet
	Diagnostics diag.Diagnostics
}
