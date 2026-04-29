package compiler

import (
	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

type Artifact struct {
	Document    *ArtifactDocument             `json:"document,omitempty"`
	Binding     *ArtifactBinding              `json:"binding,omitempty"`
	Checks      *ArtifactChecks               `json:"checks,omitempty"`
	HIR         *ArtifactHIR                  `json:"hir,omitempty"`
	Diagnostics list.List[ArtifactDiagnostic] `json:"diagnostics"`
}

type ArtifactDocument struct {
	Forms   list.List[ArtifactForm]                     `json:"forms"`
	Symbols *mapping.OrderedMap[string, ArtifactSymbol] `json:"symbols"`
	Consts  *mapping.OrderedMap[string, ArtifactValue]  `json:"consts"`
}

type ArtifactBinding struct {
	Files     list.List[string]                             `json:"files"`
	Scopes    *mapping.OrderedMap[string, ArtifactScope]    `json:"scopes"`
	Locals    *mapping.OrderedMap[string, ArtifactLocal]    `json:"locals"`
	Uses      *mapping.OrderedMap[string, ArtifactUse]      `json:"uses"`
	Symbols   *mapping.OrderedMap[string, ArtifactSymbol]   `json:"symbols"`
	Consts    *mapping.OrderedMap[string, ArtifactConst]    `json:"consts"`
	Functions *mapping.OrderedMap[string, ArtifactFunction] `json:"functions"`
}

type ArtifactChecks struct {
	Exprs  *mapping.OrderedMap[string, ArtifactExprCheck]  `json:"exprs"`
	Fields *mapping.OrderedMap[string, ArtifactFieldCheck] `json:"fields"`
	Calls  *mapping.OrderedMap[string, ArtifactCallCheck]  `json:"calls"`
}

type ArtifactHIR struct {
	Forms   list.List[ArtifactHIRForm]                    `json:"forms"`
	Symbols *mapping.OrderedMap[string, ArtifactSymbol]   `json:"symbols"`
	Consts  *mapping.OrderedMap[string, ArtifactHIRConst] `json:"consts"`
}

type ArtifactPosition struct {
	Offset int `json:"offset"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

type ArtifactSpan struct {
	Path  string           `json:"path,omitempty"`
	Start ArtifactPosition `json:"start"`
	End   ArtifactPosition `json:"end"`
}

type ArtifactDiagnostic struct {
	Severity diag.Severity `json:"severity"`
	Message  string        `json:"message"`
	Span     ArtifactSpan  `json:"span"`
}

type ArtifactSymbol struct {
	Name string       `json:"name"`
	Kind string       `json:"kind"`
	Span ArtifactSpan `json:"span"`
}

type ArtifactForm struct {
	Kind   string                                     `json:"kind"`
	Label  *FormLabel                                 `json:"label,omitempty"`
	Symbol *ArtifactSymbol                            `json:"symbol,omitempty"`
	Fields *mapping.OrderedMap[string, ArtifactValue] `json:"fields"`
	Forms  list.List[ArtifactForm]                    `json:"forms"`
	Calls  list.List[ArtifactCall]                    `json:"calls"`
	Span   ArtifactSpan                               `json:"span"`
}

type ArtifactCall struct {
	Name string                   `json:"name"`
	Args list.List[ArtifactValue] `json:"args"`
	Span ArtifactSpan             `json:"span"`
}

type ArtifactScope struct {
	ID       string       `json:"id"`
	Kind     ScopeKind    `json:"kind"`
	FormKind string       `json:"formKind,omitempty"`
	ParentID string       `json:"parentId,omitempty"`
	Span     ArtifactSpan `json:"span"`
}

type ArtifactConst struct {
	Name string       `json:"name"`
	Type ArtifactType `json:"type"`
	Span ArtifactSpan `json:"span"`
}

type ArtifactLocal struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Kind    LocalBindingKind `json:"kind"`
	ScopeID string           `json:"scopeId"`
	Type    ArtifactType     `json:"type"`
	Span    ArtifactSpan     `json:"span"`
}

type ArtifactFunction struct {
	Name   string                   `json:"name"`
	Params list.List[ArtifactParam] `json:"params"`
	Result ArtifactType             `json:"result"`
	Span   ArtifactSpan             `json:"span"`
}

type ArtifactParam struct {
	Name string       `json:"name"`
	Type ArtifactType `json:"type"`
	Span ArtifactSpan `json:"span"`
}

type ArtifactUse struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Kind     NameUseKind  `json:"kind"`
	ScopeID  string       `json:"scopeId,omitempty"`
	TargetID string       `json:"targetId,omitempty"`
	Span     ArtifactSpan `json:"span"`
}

type ArtifactExprCheck struct {
	ID      string       `json:"id"`
	Kind    string       `json:"kind"`
	ScopeID string       `json:"scopeId"`
	Type    ArtifactType `json:"type"`
	Span    ArtifactSpan `json:"span"`
}

type ArtifactFieldCheck struct {
	ID       string       `json:"id"`
	FormKind string       `json:"formKind"`
	Field    string       `json:"field"`
	ScopeID  string       `json:"scopeId"`
	Expected ArtifactType `json:"expected"`
	Actual   ArtifactType `json:"actual"`
	Span     ArtifactSpan `json:"span"`
}

type ArtifactCallCheck struct {
	ID      string                  `json:"id"`
	Name    string                  `json:"name"`
	ScopeID string                  `json:"scopeId"`
	Args    list.List[ArtifactType] `json:"args"`
	Result  ArtifactType            `json:"result"`
	Span    ArtifactSpan            `json:"span"`
}

type ArtifactHIRConst struct {
	Name  string        `json:"name"`
	Type  ArtifactType  `json:"type"`
	Value ArtifactValue `json:"value"`
	Span  ArtifactSpan  `json:"span"`
}

type ArtifactHIRForm struct {
	Kind    string                                        `json:"kind"`
	ScopeID string                                        `json:"scopeId,omitempty"`
	Label   *FormLabel                                    `json:"label,omitempty"`
	Symbol  *ArtifactSymbol                               `json:"symbol,omitempty"`
	Fields  *mapping.OrderedMap[string, ArtifactHIRField] `json:"fields"`
	Forms   list.List[ArtifactHIRForm]                    `json:"forms"`
	Calls   list.List[ArtifactHIRCall]                    `json:"calls"`
	Span    ArtifactSpan                                  `json:"span"`
}

type ArtifactHIRField struct {
	Name     string        `json:"name"`
	ScopeID  string        `json:"scopeId,omitempty"`
	Expected ArtifactType  `json:"expected"`
	Actual   ArtifactType  `json:"actual"`
	Value    ArtifactValue `json:"value"`
	Span     ArtifactSpan  `json:"span"`
}

type ArtifactHIRCall struct {
	Name    string                    `json:"name"`
	ScopeID string                    `json:"scopeId,omitempty"`
	Args    list.List[ArtifactHIRArg] `json:"args"`
	Result  ArtifactType              `json:"result"`
	Span    ArtifactSpan              `json:"span"`
}

type ArtifactHIRArg struct {
	Type  ArtifactType  `json:"type"`
	Value ArtifactValue `json:"value"`
}

type ArtifactType struct {
	Kind string        `json:"kind"`
	Name string        `json:"name,omitempty"`
	Elem *ArtifactType `json:"elem,omitempty"`
}

type ArtifactValue struct {
	Kind     string                                     `json:"kind"`
	String   string                                     `json:"string,omitempty"`
	Int      int64                                      `json:"int,omitempty"`
	Float    float64                                    `json:"float,omitempty"`
	Bool     bool                                       `json:"bool,omitempty"`
	Ref      *schema.Ref                                `json:"ref,omitempty"`
	Duration *schema.Duration                           `json:"duration,omitempty"`
	Size     *schema.Size                               `json:"size,omitempty"`
	Items    list.List[ArtifactValue]                   `json:"items"`
	Fields   *mapping.OrderedMap[string, ArtifactValue] `json:"fields"`
}
