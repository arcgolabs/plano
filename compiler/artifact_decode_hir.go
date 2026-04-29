package compiler

import (
	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
)

func (h *ArtifactHIR) hir() (*HIR, error) {
	out := emptyHIR()
	if h == nil {
		return out, nil
	}
	out.Symbols = decodeArtifactSymbolMap(h.Symbols)
	if err := decodeHIRConsts(out, h.Consts); err != nil {
		return nil, err
	}
	forms, err := decodeArtifactList(h.Forms, artifactHIRFormToHIR)
	if err != nil {
		return nil, err
	}
	out.Forms = forms
	return out, nil
}

func decodeHIRConsts(out *HIR, consts *mapping.OrderedMap[string, ArtifactHIRConst]) error {
	if out == nil || consts == nil {
		return nil
	}
	for _, name := range consts.Keys() {
		item, _ := consts.Get(name)
		typ, err := item.Type.Type()
		if err != nil {
			return err
		}
		value, err := item.Value.Value()
		if err != nil {
			return err
		}
		out.Consts.Set(name, HIRConst{
			Name:  item.Name,
			Type:  typ,
			Value: value,
		})
	}
	return nil
}

func artifactHIRFormToHIR(item ArtifactHIRForm) (HIRForm, error) {
	fields, err := decodeHIRFields(item.Fields)
	if err != nil {
		return HIRForm{}, err
	}
	forms, err := decodeArtifactList(item.Forms, artifactHIRFormToHIR)
	if err != nil {
		return HIRForm{}, err
	}
	calls, err := decodeHIRCalls(item.Calls)
	if err != nil {
		return HIRForm{}, err
	}
	return HIRForm{
		Kind:    item.Kind,
		ScopeID: item.ScopeID,
		Label:   item.Label,
		Symbol:  item.Symbol.symbolPtr(),
		Fields:  fields,
		Forms:   forms,
		Calls:   calls,
	}, nil
}

func decodeHIRFields(items *mapping.OrderedMap[string, ArtifactHIRField]) (*mapping.OrderedMap[string, HIRField], error) {
	out := mapping.NewOrderedMap[string, HIRField]()
	if items == nil {
		return out, nil
	}
	for _, name := range items.Keys() {
		item, _ := items.Get(name)
		expected, err := item.Expected.Type()
		if err != nil {
			return nil, err
		}
		actual, err := item.Actual.Type()
		if err != nil {
			return nil, err
		}
		value, err := item.Value.Value()
		if err != nil {
			return nil, err
		}
		out.Set(name, HIRField{
			Name:     item.Name,
			ScopeID:  item.ScopeID,
			Expected: expected,
			Actual:   actual,
			Value:    value,
		})
	}
	return out, nil
}

func decodeHIRCalls(items list.List[ArtifactHIRCall]) (list.List[HIRCall], error) {
	out := list.NewListWithCapacity[HIRCall](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		args, err := decodeHIRArgs(item.Args)
		if err != nil {
			return list.List[HIRCall]{}, err
		}
		result, err := item.Result.Type()
		if err != nil {
			return list.List[HIRCall]{}, err
		}
		out.Add(HIRCall{
			Name:    item.Name,
			ScopeID: item.ScopeID,
			Args:    args,
			Result:  result,
		})
	}
	return *out, nil
}

func decodeHIRArgs(items list.List[ArtifactHIRArg]) (list.List[HIRArg], error) {
	out := list.NewListWithCapacity[HIRArg](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		typ, err := item.Type.Type()
		if err != nil {
			return list.List[HIRArg]{}, err
		}
		value, err := item.Value.Value()
		if err != nil {
			return list.List[HIRArg]{}, err
		}
		out.Add(HIRArg{Type: typ, Value: value})
	}
	return *out, nil
}
