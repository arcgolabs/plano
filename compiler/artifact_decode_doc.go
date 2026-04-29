package compiler

import (
	"go/token"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
)

func (d *ArtifactDocument) document() (*Document, error) {
	forms, err := d.forms()
	if err != nil {
		return nil, err
	}
	consts, err := d.consts()
	if err != nil {
		return nil, err
	}
	return &Document{
		Forms:   forms,
		Symbols: decodeArtifactSymbolMap(d.Symbols),
		Consts:  consts,
	}, nil
}

func (d *ArtifactDocument) forms() (list.List[Form], error) {
	if d == nil {
		return list.List[Form]{}, nil
	}
	return decodeArtifactList(d.Forms, artifactFormToDocument)
}

func (d *ArtifactDocument) consts() (*mapping.OrderedMap[string, any], error) {
	if d == nil {
		return mapping.NewOrderedMap[string, any](), nil
	}
	return decodeArtifactMapToOrdered(d.Consts, func(item ArtifactValue) (any, error) {
		return item.Value()
	})
}

func artifactFormToDocument(item ArtifactForm) (Form, error) {
	fields, err := decodeArtifactMapToOrdered(item.Fields, func(value ArtifactValue) (any, error) {
		return value.Value()
	})
	if err != nil {
		return Form{}, err
	}
	forms, err := decodeArtifactList(item.Forms, artifactFormToDocument)
	if err != nil {
		return Form{}, err
	}
	calls, err := decodeArtifactList(item.Calls, artifactCallToDocument)
	if err != nil {
		return Form{}, err
	}
	return Form{
		Kind:   item.Kind,
		Label:  item.Label,
		Symbol: item.Symbol.symbolPtr(),
		Fields: fields,
		Forms:  forms,
		Calls:  calls,
		Pos:    token.NoPos,
		End:    token.NoPos,
	}, nil
}

func artifactCallToDocument(item ArtifactCall) (Call, error) {
	args, err := decodeArtifactListToAny(item.Args)
	if err != nil {
		return Call{}, err
	}
	return Call{Name: item.Name, Args: args}, nil
}

func decodeArtifactListToAny(items list.List[ArtifactValue]) (list.List[any], error) {
	out := list.NewListWithCapacity[any](items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		value, err := item.Value()
		if err != nil {
			return list.List[any]{}, err
		}
		out.Add(value)
	}
	return *out, nil
}

func decodeArtifactSymbolMap(items *mapping.OrderedMap[string, ArtifactSymbol]) *mapping.OrderedMap[string, Symbol] {
	out := mapping.NewOrderedMap[string, Symbol]()
	if items == nil {
		return out
	}
	for _, name := range items.Keys() {
		item, _ := items.Get(name)
		out.Set(name, item.symbol())
	}
	return out
}
