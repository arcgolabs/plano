package compiler

import (
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

type artifactEntry[T any] struct {
	Key   string `json:"key"`
	Value T      `json:"value"`
}

type artifactWire struct {
	SchemaVersion string                   `json:"schemaVersion"`
	Document      *artifactDocumentWire    `json:"document,omitempty"`
	Binding       *artifactBindingWire     `json:"binding,omitempty"`
	Checks        *artifactChecksWire      `json:"checks,omitempty"`
	HIR           *artifactHIRWire         `json:"hir,omitempty"`
	Diagnostics   []artifactDiagnosticWire `json:"diagnostics"`
}

type artifactDiagnosticWire struct {
	Severity diag.Severity                    `json:"severity"`
	Code     diag.Code                        `json:"code,omitempty"`
	Message  string                           `json:"message"`
	Span     ArtifactSpan                     `json:"span"`
	Related  []artifactRelatedInformationWire `json:"related"`
}

type artifactRelatedInformationWire struct {
	Message string       `json:"message"`
	Span    ArtifactSpan `json:"span"`
}

type artifactDocumentWire struct {
	Forms   []artifactFormWire                 `json:"forms"`
	Symbols []artifactEntry[ArtifactSymbol]    `json:"symbols"`
	Consts  []artifactEntry[artifactValueWire] `json:"consts"`
}

type artifactBindingWire struct {
	Files     []string                              `json:"files"`
	Scopes    []artifactEntry[ArtifactScope]        `json:"scopes"`
	Locals    []artifactEntry[ArtifactLocal]        `json:"locals"`
	Uses      []artifactEntry[ArtifactUse]          `json:"uses"`
	Symbols   []artifactEntry[ArtifactSymbol]       `json:"symbols"`
	Consts    []artifactEntry[ArtifactConst]        `json:"consts"`
	Functions []artifactEntry[artifactFunctionWire] `json:"functions"`
}

type artifactChecksWire struct {
	Exprs  []artifactEntry[ArtifactExprCheck]     `json:"exprs"`
	Fields []artifactEntry[ArtifactFieldCheck]    `json:"fields"`
	Calls  []artifactEntry[artifactCallCheckWire] `json:"calls"`
}

type artifactHIRWire struct {
	Forms   []artifactHIRFormWire                 `json:"forms"`
	Symbols []artifactEntry[ArtifactSymbol]       `json:"symbols"`
	Consts  []artifactEntry[artifactHIRConstWire] `json:"consts"`
}

type artifactFormWire struct {
	Kind   string                             `json:"kind"`
	Label  *FormLabel                         `json:"label,omitempty"`
	Symbol *ArtifactSymbol                    `json:"symbol,omitempty"`
	Fields []artifactEntry[artifactValueWire] `json:"fields"`
	Forms  []artifactFormWire                 `json:"forms"`
	Calls  []artifactCallWire                 `json:"calls"`
	Span   ArtifactSpan                       `json:"span"`
}

type artifactCallWire struct {
	Name string              `json:"name"`
	Args []artifactValueWire `json:"args"`
	Span ArtifactSpan        `json:"span"`
}

type artifactFunctionWire struct {
	Name   string          `json:"name"`
	Params []ArtifactParam `json:"params"`
	Result ArtifactType    `json:"result"`
	Span   ArtifactSpan    `json:"span"`
}

type artifactCallCheckWire struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	ScopeID string         `json:"scopeId"`
	Args    []ArtifactType `json:"args"`
	Result  ArtifactType   `json:"result"`
	Span    ArtifactSpan   `json:"span"`
}

type artifactHIRConstWire struct {
	Name  string            `json:"name"`
	Type  ArtifactType      `json:"type"`
	Value artifactValueWire `json:"value"`
	Span  ArtifactSpan      `json:"span"`
}

type artifactHIRFormWire struct {
	Kind    string                                `json:"kind"`
	ScopeID string                                `json:"scopeId,omitempty"`
	Label   *FormLabel                            `json:"label,omitempty"`
	Symbol  *ArtifactSymbol                       `json:"symbol,omitempty"`
	Fields  []artifactEntry[artifactHIRFieldWire] `json:"fields"`
	Forms   []artifactHIRFormWire                 `json:"forms"`
	Calls   []artifactHIRCallWire                 `json:"calls"`
	Span    ArtifactSpan                          `json:"span"`
}

type artifactHIRFieldWire struct {
	Name     string            `json:"name"`
	ScopeID  string            `json:"scopeId,omitempty"`
	Expected ArtifactType      `json:"expected"`
	Actual   ArtifactType      `json:"actual"`
	Value    artifactValueWire `json:"value"`
	Span     ArtifactSpan      `json:"span"`
}

type artifactHIRCallWire struct {
	Name    string               `json:"name"`
	ScopeID string               `json:"scopeId,omitempty"`
	Args    []artifactHIRArgWire `json:"args"`
	Result  ArtifactType         `json:"result"`
	Span    ArtifactSpan         `json:"span"`
}

type artifactHIRArgWire struct {
	Type  ArtifactType      `json:"type"`
	Value artifactValueWire `json:"value"`
}

type artifactValueWire struct {
	Kind     string                             `json:"kind"`
	String   string                             `json:"string,omitempty"`
	Int      int64                              `json:"int,omitempty"`
	Float    float64                            `json:"float,omitempty"`
	Bool     bool                               `json:"bool,omitempty"`
	Ref      *schema.Ref                        `json:"ref,omitempty"`
	Duration *schema.Duration                   `json:"duration,omitempty"`
	Size     *schema.Size                       `json:"size,omitempty"`
	Items    []artifactValueWire                `json:"items"`
	Fields   []artifactEntry[artifactValueWire] `json:"fields"`
}
