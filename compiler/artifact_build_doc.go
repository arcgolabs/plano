package compiler

import (
	"fmt"
	"go/token"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/diag"
)

type artifactBuilder struct {
	fset *token.FileSet
}

func (b artifactBuilder) build(result Result) (*Artifact, error) {
	document, err := b.document(result.Document)
	if err != nil {
		return nil, err
	}
	binding, err := b.binding(result.Binding)
	if err != nil {
		return nil, err
	}
	checks := b.checks(result.Checks)
	hir, err := b.hir(result.HIR)
	if err != nil {
		return nil, err
	}
	return &Artifact{
		SchemaVersion: ArtifactSchemaVersion,
		Document:      document,
		Binding:       binding,
		Checks:        checks,
		HIR:           hir,
		Diagnostics:   b.diagnostics(result.Diagnostics),
	}, nil
}

func (b artifactBuilder) diagnostics(items diag.Diagnostics) list.List[ArtifactDiagnostic] {
	out := list.NewListWithCapacity[ArtifactDiagnostic](len(items))
	for index := range items {
		item := items[index]
		out.Add(ArtifactDiagnostic{
			Severity: item.Severity,
			Code:     item.Code,
			Message:  item.Message,
			Span:     b.span(item.Pos, item.End),
			Related:  b.relatedDiagnostics(item.Related),
		})
	}
	return *out
}

func (b artifactBuilder) relatedDiagnostics(items list.List[diag.RelatedInformation]) list.List[ArtifactRelatedInformation] {
	out := list.NewListWithCapacity[ArtifactRelatedInformation](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		out.Add(ArtifactRelatedInformation{
			Message: item.Message,
			Span:    b.span(item.Pos, item.End),
		})
	}
	return *out
}

func (b artifactBuilder) document(doc *Document) (*ArtifactDocument, error) {
	if doc == nil {
		return emptyArtifactDocument(), nil
	}
	forms, err := b.forms(doc.Forms)
	if err != nil {
		return nil, err
	}
	consts, err := b.valueMap(doc.Consts)
	if err != nil {
		return nil, err
	}
	return &ArtifactDocument{
		Forms:   forms,
		Symbols: b.symbolMap(doc.Symbols),
		Consts:  consts,
	}, nil
}

func (b artifactBuilder) forms(items list.List[Form]) (list.List[ArtifactForm], error) {
	return encodeArtifactList(items, b.form)
}

func (b artifactBuilder) form(item Form) (ArtifactForm, error) {
	fields, err := b.valueMap(item.Fields)
	if err != nil {
		return ArtifactForm{}, err
	}
	forms, err := b.forms(item.Forms)
	if err != nil {
		return ArtifactForm{}, err
	}
	calls, err := b.calls(item.Calls)
	if err != nil {
		return ArtifactForm{}, err
	}
	return ArtifactForm{
		Kind:   item.Kind,
		Label:  item.Label,
		Symbol: b.symbolPtr(item.Symbol),
		Fields: fields,
		Forms:  forms,
		Calls:  calls,
		Span:   b.span(item.Pos, item.End),
	}, nil
}

func (b artifactBuilder) calls(items list.List[Call]) (list.List[ArtifactCall], error) {
	out := list.NewListWithCapacity[ArtifactCall](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		args, err := b.values(item.Args)
		if err != nil {
			return list.List[ArtifactCall]{}, err
		}
		out.Add(ArtifactCall{
			Name: item.Name,
			Args: args,
			Span: b.span(item.Pos, item.End),
		})
	}
	return *out, nil
}

func (b artifactBuilder) values(items list.List[any]) (list.List[ArtifactValue], error) {
	return encodeArtifactList(items, artifactValue)
}

func (b artifactBuilder) valueMap(items *mapping.OrderedMap[string, any]) (*mapping.OrderedMap[string, ArtifactValue], error) {
	return encodeArtifactMapToOrdered(items, artifactValue)
}

func (b artifactBuilder) symbolMap(items *mapping.OrderedMap[string, Symbol]) *mapping.OrderedMap[string, ArtifactSymbol] {
	out := mapping.NewOrderedMap[string, ArtifactSymbol]()
	if items == nil {
		return out
	}
	for _, name := range items.Keys() {
		item, _ := items.Get(name)
		out.Set(name, ArtifactSymbol{
			Name: item.Name,
			Kind: item.Kind,
			Span: b.span(item.Pos, item.End),
		})
	}
	return out
}

func (b artifactBuilder) symbolPtr(item *Symbol) *ArtifactSymbol {
	if item == nil {
		return nil
	}
	return &ArtifactSymbol{
		Name: item.Name,
		Kind: item.Kind,
		Span: b.span(item.Pos, item.End),
	}
}

func (b artifactBuilder) span(pos, end token.Pos) ArtifactSpan {
	if b.fset == nil || !pos.IsValid() {
		return ArtifactSpan{}
	}
	start := b.fset.Position(pos)
	finish := start
	if end.IsValid() && end >= pos {
		finish = b.fset.Position(end)
	}
	return ArtifactSpan{
		Path: start.Filename,
		Start: ArtifactPosition{
			Offset: start.Offset,
			Line:   start.Line,
			Column: start.Column,
		},
		End: ArtifactPosition{
			Offset: finish.Offset,
			Line:   finish.Line,
			Column: finish.Column,
		},
	}
}

func encodeArtifactList[T any, W any](items list.List[T], encode func(T) (W, error)) (list.List[W], error) {
	if encode == nil {
		return list.List[W]{}, errNilArtifactListCodec
	}
	out := list.NewListWithCapacity[W](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		value, err := encode(item)
		if err != nil {
			return list.List[W]{}, err
		}
		out.Add(value)
	}
	return *out, nil
}

func encodeArtifactMapToOrdered[V any, W any](items *mapping.OrderedMap[string, V], encode func(V) (W, error)) (*mapping.OrderedMap[string, W], error) {
	if encode == nil {
		return nil, errNilArtifactMapCodec
	}
	out := mapping.NewOrderedMap[string, W]()
	if items == nil {
		return out, nil
	}
	for _, key := range items.Keys() {
		item, _ := items.Get(key)
		value, err := encode(item)
		if err != nil {
			return nil, err
		}
		out.Set(key, value)
	}
	return out, nil
}

func artifactUnknownValueError(value any) error {
	return fmt.Errorf("artifact value: unsupported %T", value)
}
