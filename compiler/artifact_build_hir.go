package compiler

import (
	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
)

func (b artifactBuilder) hir(hir *HIR) (*ArtifactHIR, error) {
	if hir == nil {
		return emptyArtifactHIR(), nil
	}
	out := emptyArtifactHIR()
	out.Symbols = b.symbolMap(hir.Symbols)
	if err := b.addHIRConsts(out, hir); err != nil {
		return nil, err
	}
	forms, err := b.hirForms(hir.Forms)
	if err != nil {
		return nil, err
	}
	out.Forms = forms
	return out, nil
}

func (b artifactBuilder) addHIRConsts(out *ArtifactHIR, hir *HIR) error {
	if out == nil || hir == nil || hir.Consts == nil {
		return nil
	}
	for _, name := range hir.Consts.Keys() {
		item, _ := hir.Consts.Get(name)
		value, err := artifactValue(item.Value)
		if err != nil {
			return err
		}
		out.Consts.Set(name, ArtifactHIRConst{
			Name:  item.Name,
			Type:  artifactType(item.Type),
			Value: value,
			Span:  b.span(item.Pos, item.End),
		})
	}
	return nil
}

func (b artifactBuilder) hirForms(items list.List[HIRForm]) (list.List[ArtifactHIRForm], error) {
	return encodeArtifactList(items, b.hirForm)
}

func (b artifactBuilder) hirForm(item HIRForm) (ArtifactHIRForm, error) {
	fields, err := b.hirFieldMap(item.Fields)
	if err != nil {
		return ArtifactHIRForm{}, err
	}
	forms, err := b.hirForms(item.Forms)
	if err != nil {
		return ArtifactHIRForm{}, err
	}
	calls, err := b.hirCalls(item.Calls)
	if err != nil {
		return ArtifactHIRForm{}, err
	}
	return ArtifactHIRForm{
		Kind:    item.Kind,
		ScopeID: item.ScopeID,
		Label:   item.Label,
		Symbol:  b.symbolPtr(item.Symbol),
		Fields:  fields,
		Forms:   forms,
		Calls:   calls,
		Span:    b.span(item.Pos, item.End),
	}, nil
}

func (b artifactBuilder) hirFieldMap(items *mapping.OrderedMap[string, HIRField]) (*mapping.OrderedMap[string, ArtifactHIRField], error) {
	out := mapping.NewOrderedMap[string, ArtifactHIRField]()
	if items == nil {
		return out, nil
	}
	for _, name := range items.Keys() {
		field, _ := items.Get(name)
		value, err := artifactValue(field.Value)
		if err != nil {
			return nil, err
		}
		out.Set(name, ArtifactHIRField{
			Name:     field.Name,
			ScopeID:  field.ScopeID,
			Expected: artifactType(field.Expected),
			Actual:   artifactType(field.Actual),
			Value:    value,
			Span:     b.span(field.Pos, field.End),
		})
	}
	return out, nil
}

func (b artifactBuilder) hirCalls(items list.List[HIRCall]) (list.List[ArtifactHIRCall], error) {
	out := list.NewListWithCapacity[ArtifactHIRCall](items.Len())
	for index := range items.Len() {
		call, _ := items.Get(index)
		args, err := b.hirArgs(call.Args)
		if err != nil {
			return list.List[ArtifactHIRCall]{}, err
		}
		out.Add(ArtifactHIRCall{
			Name:    call.Name,
			ScopeID: call.ScopeID,
			Args:    args,
			Result:  artifactType(call.Result),
			Span:    b.span(call.Pos, call.End),
		})
	}
	return *out, nil
}

func (b artifactBuilder) hirArgs(items list.List[HIRArg]) (list.List[ArtifactHIRArg], error) {
	out := list.NewListWithCapacity[ArtifactHIRArg](items.Len())
	for index := range items.Len() {
		arg, _ := items.Get(index)
		value, err := artifactValue(arg.Value)
		if err != nil {
			return list.List[ArtifactHIRArg]{}, err
		}
		out.Add(ArtifactHIRArg{
			Type:  artifactType(arg.Type),
			Value: value,
		})
	}
	return *out, nil
}
