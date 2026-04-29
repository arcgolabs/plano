package compiler

import (
	"context"
	"encoding/json"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/diag"
)

func ArtifactFromResult(result Result) (*Artifact, error) {
	builder := artifactBuilder{fset: result.FileSet}
	return builder.build(result)
}

func (r Result) Artifact() (*Artifact, error) {
	return ArtifactFromResult(r)
}

func (a Artifact) MarshalBinary() ([]byte, error) {
	return a.MarshalJSON()
}

func (a *Artifact) UnmarshalBinary(data []byte) error {
	return a.UnmarshalJSON(data)
}

func (a Artifact) MarshalJSON() ([]byte, error) {
	wire, err := a.wire()
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(wire)
	if err != nil {
		return nil, errWrapArtifactJSON("marshal artifact json", err)
	}
	return data, nil
}

func (a *Artifact) UnmarshalJSON(data []byte) error {
	if a == nil {
		return errNilArtifactReceiver
	}
	var wire artifactWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return errWrapArtifactJSON("unmarshal artifact json", err)
	}
	value, err := wire.artifact()
	if err != nil {
		return err
	}
	*a = value
	return nil
}

func (a Artifact) Result() (Result, error) {
	document, err := a.DocumentValue()
	if err != nil {
		return Result{}, err
	}
	binding, err := a.BindingValue()
	if err != nil {
		return Result{}, err
	}
	checks, err := a.ChecksValue()
	if err != nil {
		return Result{}, err
	}
	hir, err := a.HIRValue()
	if err != nil {
		return Result{}, err
	}
	return Result{
		Document:    document,
		Binding:     binding,
		Checks:      checks,
		HIR:         hir,
		Diagnostics: a.DiagnosticsValue(),
	}, nil
}

func (a Artifact) DocumentValue() (*Document, error) {
	if a.Document == nil {
		return emptyDocument(), nil
	}
	return a.Document.document()
}

func (a Artifact) BindingValue() (*Binding, error) {
	if a.Binding == nil {
		return emptyBinding(), nil
	}
	return a.Binding.binding()
}

func (a Artifact) ChecksValue() (*CheckInfo, error) {
	if a.Checks == nil {
		return emptyChecks(), nil
	}
	return a.Checks.checks()
}

func (a Artifact) HIRValue() (*HIR, error) {
	if a.HIR == nil {
		return emptyHIR(), nil
	}
	return a.HIR.hir()
}

func (a Artifact) DiagnosticsValue() diag.Diagnostics {
	out := make(diag.Diagnostics, 0, a.Diagnostics.Len())
	for index := range a.Diagnostics.Len() {
		item, _ := a.Diagnostics.Get(index)
		out = append(out, diag.Diagnostic{
			Severity: item.Severity,
			Message:  item.Message,
		})
	}
	return out
}

func (c *Compiler) CompileSourceArtifact(ctx context.Context, filename string, src []byte) (*Artifact, error) {
	return c.CompileSourceDetailed(ctx, filename, src).Artifact()
}

func (c *Compiler) CompileFileArtifact(ctx context.Context, filename string) (*Artifact, error) {
	return c.CompileFileDetailed(ctx, filename).Artifact()
}

func (c *Compiler) CompileStringArtifact(ctx context.Context, filename, src string) (*Artifact, error) {
	return c.CompileSourceArtifact(ctx, filename, []byte(src))
}

func emptyArtifactDocument() *ArtifactDocument {
	return &ArtifactDocument{
		Forms:   list.List[ArtifactForm]{},
		Symbols: mapping.NewOrderedMap[string, ArtifactSymbol](),
		Consts:  mapping.NewOrderedMap[string, ArtifactValue](),
	}
}

func emptyArtifactBinding() *ArtifactBinding {
	return &ArtifactBinding{
		Files:     list.List[string]{},
		Scopes:    mapping.NewOrderedMap[string, ArtifactScope](),
		Locals:    mapping.NewOrderedMap[string, ArtifactLocal](),
		Uses:      mapping.NewOrderedMap[string, ArtifactUse](),
		Symbols:   mapping.NewOrderedMap[string, ArtifactSymbol](),
		Consts:    mapping.NewOrderedMap[string, ArtifactConst](),
		Functions: mapping.NewOrderedMap[string, ArtifactFunction](),
	}
}

func emptyArtifactChecks() *ArtifactChecks {
	return &ArtifactChecks{
		Exprs:  mapping.NewOrderedMap[string, ArtifactExprCheck](),
		Fields: mapping.NewOrderedMap[string, ArtifactFieldCheck](),
		Calls:  mapping.NewOrderedMap[string, ArtifactCallCheck](),
	}
}

func emptyArtifactHIR() *ArtifactHIR {
	return &ArtifactHIR{
		Forms:   list.List[ArtifactHIRForm]{},
		Symbols: mapping.NewOrderedMap[string, ArtifactSymbol](),
		Consts:  mapping.NewOrderedMap[string, ArtifactHIRConst](),
	}
}

func emptyDocument() *Document {
	return &Document{
		Forms:   list.List[Form]{},
		Symbols: mapping.NewOrderedMap[string, Symbol](),
		Consts:  mapping.NewOrderedMap[string, any](),
	}
}

func emptyBinding() *Binding {
	return &Binding{
		Files:     list.List[string]{},
		Scopes:    mapping.NewOrderedMap[string, ScopeBinding](),
		Locals:    mapping.NewOrderedMap[string, LocalBinding](),
		Uses:      mapping.NewOrderedMap[string, NameUse](),
		Symbols:   mapping.NewOrderedMap[string, Symbol](),
		Consts:    mapping.NewOrderedMap[string, ConstBinding](),
		Functions: mapping.NewOrderedMap[string, FunctionBinding](),
	}
}

func emptyChecks() *CheckInfo {
	return &CheckInfo{
		Exprs:  mapping.NewOrderedMap[string, ExprCheck](),
		Fields: mapping.NewOrderedMap[string, FieldCheck](),
		Calls:  mapping.NewOrderedMap[string, CallCheck](),
	}
}

func emptyHIR() *HIR {
	return &HIR{
		Forms:   list.List[HIRForm]{},
		Symbols: mapping.NewOrderedMap[string, Symbol](),
		Consts:  mapping.NewOrderedMap[string, HIRConst](),
	}
}
