package compiler

import (
	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
)

func artifactValueToWire(value ArtifactValue) (artifactValueWire, error) {
	items, err := encodeArtifactWireList(value.Items, artifactValueToWire)
	if err != nil {
		return artifactValueWire{}, err
	}
	fields, err := encodeArtifactWireMap(value.Fields, artifactValueToWire)
	if err != nil {
		return artifactValueWire{}, err
	}
	return artifactValueWire{
		Kind:     value.Kind,
		String:   value.String,
		Int:      value.Int,
		Float:    value.Float,
		Bool:     value.Bool,
		Ref:      value.Ref,
		Duration: value.Duration,
		Size:     value.Size,
		Items:    items,
		Fields:   fields,
	}, nil
}

func artifactValueFromWire(w artifactValueWire) (ArtifactValue, error) {
	items, err := decodeArtifactWireList(w.Items, artifactValueFromWire)
	if err != nil {
		return ArtifactValue{}, err
	}
	fields, err := decodeArtifactWireMap(w.Fields, artifactValueFromWire)
	if err != nil {
		return ArtifactValue{}, err
	}
	return ArtifactValue{
		Kind:     w.Kind,
		String:   w.String,
		Int:      w.Int,
		Float:    w.Float,
		Bool:     w.Bool,
		Ref:      w.Ref,
		Duration: w.Duration,
		Size:     w.Size,
		Items:    items,
		Fields:   fields,
	}, nil
}

func identityArtifactSymbol(value ArtifactSymbol) (ArtifactSymbol, error) { return value, nil }
func identityArtifactScope(value ArtifactScope) (ArtifactScope, error)    { return value, nil }
func identityArtifactLocal(value ArtifactLocal) (ArtifactLocal, error)    { return value, nil }
func identityArtifactUse(value ArtifactUse) (ArtifactUse, error)          { return value, nil }
func identityArtifactConst(value ArtifactConst) (ArtifactConst, error)    { return value, nil }
func identityArtifactExprCheck(value ArtifactExprCheck) (ArtifactExprCheck, error) {
	return value, nil
}
func identityArtifactFieldCheck(value ArtifactFieldCheck) (ArtifactFieldCheck, error) {
	return value, nil
}

func encodeArtifactWireList[T any, W any](items list.List[T], encode func(T) (W, error)) ([]W, error) {
	if encode == nil {
		return nil, errNilArtifactListCodec
	}
	out := make([]W, 0, items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		value, err := encode(item)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

func decodeArtifactWireList[W any, T any](items []W, decode func(W) (T, error)) (list.List[T], error) {
	if decode == nil {
		return list.List[T]{}, errNilArtifactListCodec
	}
	out := list.NewListWithCapacity[T](len(items))
	for _, item := range items {
		value, err := decode(item)
		if err != nil {
			return list.List[T]{}, err
		}
		out.Add(value)
	}
	return *out, nil
}

func encodeArtifactWireMap[V any, W any](items *mapping.OrderedMap[string, V], encode func(V) (W, error)) ([]artifactEntry[W], error) {
	if encode == nil {
		return nil, errNilArtifactMapCodec
	}
	if items == nil {
		return []artifactEntry[W]{}, nil
	}
	out := make([]artifactEntry[W], 0, items.Len())
	for _, key := range items.Keys() {
		item, _ := items.Get(key)
		value, err := encode(item)
		if err != nil {
			return nil, err
		}
		out = append(out, artifactEntry[W]{Key: key, Value: value})
	}
	return out, nil
}

func decodeArtifactWireMap[W any, V any](items []artifactEntry[W], decode func(W) (V, error)) (*mapping.OrderedMap[string, V], error) {
	if decode == nil {
		return nil, errNilArtifactMapCodec
	}
	out := mapping.NewOrderedMapWithCapacity[string, V](len(items))
	for _, item := range items {
		value, err := decode(item.Value)
		if err != nil {
			return nil, err
		}
		out.Set(item.Key, value)
	}
	return out, nil
}
