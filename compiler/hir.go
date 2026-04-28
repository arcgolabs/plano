package compiler

import (
	"go/token"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/schema"
)

type HIR struct {
	Forms   list.List[HIRForm]                    `json:"forms"   yaml:"forms"`
	Symbols *mapping.OrderedMap[string, Symbol]   `json:"symbols" yaml:"symbols"`
	Consts  *mapping.OrderedMap[string, HIRConst] `json:"consts"  yaml:"consts"`
}

type HIRConst struct {
	Name  string      `json:"name"  yaml:"name"`
	Type  schema.Type `json:"type"  yaml:"type"`
	Value any         `json:"value" yaml:"value"`
	Pos   token.Pos   `json:"pos"   yaml:"pos"`
	End   token.Pos   `json:"end"   yaml:"end"`
}

type HIRForm struct {
	Kind    string                                `json:"kind"              yaml:"kind"`
	ScopeID string                                `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Label   *FormLabel                            `json:"label,omitempty"   yaml:"label,omitempty"`
	Symbol  *Symbol                               `json:"symbol,omitempty"  yaml:"symbol,omitempty"`
	Fields  *mapping.OrderedMap[string, HIRField] `json:"fields"            yaml:"fields"`
	Forms   list.List[HIRForm]                    `json:"forms"             yaml:"forms"`
	Calls   list.List[HIRCall]                    `json:"calls"             yaml:"calls"`
	Pos     token.Pos                             `json:"pos"               yaml:"pos"`
	End     token.Pos                             `json:"end"               yaml:"end"`
}

type HIRField struct {
	Name     string      `json:"name"              yaml:"name"`
	ScopeID  string      `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Expected schema.Type `json:"expected"          yaml:"expected"`
	Actual   schema.Type `json:"actual"            yaml:"actual"`
	Value    any         `json:"value"             yaml:"value"`
	Pos      token.Pos   `json:"pos"               yaml:"pos"`
	End      token.Pos   `json:"end"               yaml:"end"`
}

type HIRCall struct {
	Name    string            `json:"name"              yaml:"name"`
	ScopeID string            `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Args    list.List[HIRArg] `json:"args"              yaml:"args"`
	Result  schema.Type       `json:"result"            yaml:"result"`
	Pos     token.Pos         `json:"pos"               yaml:"pos"`
	End     token.Pos         `json:"end"               yaml:"end"`
}

type HIRArg struct {
	Type  schema.Type `json:"type"  yaml:"type"`
	Value any         `json:"value" yaml:"value"`
}

func (d *Document) Const(name string) (any, bool) {
	if d == nil || d.Consts == nil {
		return nil, false
	}
	return d.Consts.Get(name)
}

func (d *Document) Symbol(name string) (Symbol, bool) {
	if d == nil || d.Symbols == nil {
		return Symbol{}, false
	}
	return d.Symbols.Get(name)
}

func (f *Form) Field(name string) (any, bool) {
	if f == nil || f.Fields == nil {
		return nil, false
	}
	return f.Fields.Get(name)
}

func (f *HIRForm) Field(name string) (HIRField, bool) {
	if f == nil || f.Fields == nil {
		return HIRField{}, false
	}
	return f.Fields.Get(name)
}
