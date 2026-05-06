package compiler

import (
	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/schema"
)

func artifactType(typ schema.Type) ArtifactType {
	switch current := normalizeType(typ).(type) {
	case schema.BuiltinType:
		return ArtifactType{Kind: "builtin", Name: current.String()}
	case schema.ListType:
		elem := artifactType(current.Elem)
		return ArtifactType{Kind: "list", Elem: &elem}
	case schema.MapType:
		elem := artifactType(current.Elem)
		return ArtifactType{Kind: "map", Elem: &elem}
	case schema.RefType:
		return ArtifactType{Kind: "ref", Name: current.Kind}
	case schema.NamedType:
		return ArtifactType{Kind: "named", Name: current.Name}
	default:
		return ArtifactType{Kind: "builtin", Name: schema.TypeAny.String()}
	}
}

func (t ArtifactType) Type() (schema.Type, error) {
	switch t.Kind {
	case "builtin":
		return schema.BuiltinType(t.Name), nil
	case "list":
		return artifactNestedType(schema.ListType{}, t.Elem, errMissingListElem)
	case "map":
		return artifactNestedType(schema.MapType{}, t.Elem, errMissingMapElem)
	case "ref":
		return schema.RefType{Kind: t.Name}, nil
	case "named":
		return schema.NamedType{Name: t.Name}, nil
	default:
		return nil, compilerErrorf("artifact type: unknown %q", t.Kind)
	}
}

func artifactNestedType(base any, elem *ArtifactType, missing error) (schema.Type, error) {
	if elem == nil {
		return nil, missing
	}
	value, err := elem.Type()
	if err != nil {
		return nil, err
	}
	switch base.(type) {
	case schema.ListType:
		return schema.ListType{Elem: value}, nil
	default:
		return schema.MapType{Elem: value}, nil
	}
}

func artifactValue(value any) (ArtifactValue, error) {
	if current, ok := artifactScalarValue(value); ok {
		return current, nil
	}
	switch current := value.(type) {
	case schema.Ref:
		ref := current
		return ArtifactValue{Kind: "ref", Ref: &ref}, nil
	case schema.Duration:
		duration := current
		return ArtifactValue{Kind: "duration", Duration: &duration}, nil
	case schema.Size:
		size := current
		return ArtifactValue{Kind: "size", Size: &size}, nil
	case []any:
		return artifactListValue(current)
	case *mapping.OrderedMap[string, any]:
		return artifactMapValue(current)
	case map[string]any:
		return artifactMapValue(orderedAnyMap(current))
	default:
		return ArtifactValue{}, artifactUnknownValueError(value)
	}
}

func artifactScalarValue(value any) (ArtifactValue, bool) {
	switch current := value.(type) {
	case nil, schema.Null:
		return ArtifactValue{Kind: "null"}, true
	case string:
		return ArtifactValue{Kind: "string", String: current}, true
	case int64:
		return ArtifactValue{Kind: "int", Int: current}, true
	case float64:
		return ArtifactValue{Kind: "float", Float: current}, true
	case bool:
		return ArtifactValue{Kind: "bool", Bool: current}, true
	default:
		return ArtifactValue{}, false
	}
}

func artifactListValue(items []any) (ArtifactValue, error) {
	values := make([]ArtifactValue, 0, len(items))
	for _, item := range items {
		value, err := artifactValue(item)
		if err != nil {
			return ArtifactValue{}, err
		}
		values = append(values, value)
	}
	out := ArtifactValue{Kind: "list"}
	out.Items.MergeSlice(values)
	return out, nil
}

func artifactMapValue(items *mapping.OrderedMap[string, any]) (ArtifactValue, error) {
	fields := mapping.NewOrderedMap[string, ArtifactValue]()
	if items == nil {
		return ArtifactValue{Kind: "map", Fields: fields}, nil
	}
	for _, name := range items.Keys() {
		item, _ := items.Get(name)
		value, err := artifactValue(item)
		if err != nil {
			return ArtifactValue{}, err
		}
		fields.Set(name, value)
	}
	return ArtifactValue{Kind: "map", Fields: fields}, nil
}

func (v ArtifactValue) Value() (any, error) {
	if value, ok := artifactScalarDecodedValue(v); ok {
		return value, nil
	}
	switch v.Kind {
	case "ref":
		return requiredArtifactValue(v.Ref, errMissingRefData)
	case "duration":
		return requiredArtifactValue(v.Duration, errMissingDurationData)
	case "size":
		return requiredArtifactValue(v.Size, errMissingSizeData)
	case "list":
		return decodeArtifactAnyList(v.Items)
	case "map":
		return decodeArtifactAnyMap(v.Fields)
	default:
		return nil, compilerErrorf("artifact value: unknown %q", v.Kind)
	}
}

func artifactScalarDecodedValue(value ArtifactValue) (any, bool) {
	switch value.Kind {
	case "null":
		return schema.Null{}, true
	case "string":
		return value.String, true
	case "int":
		return value.Int, true
	case "float":
		return value.Float, true
	case "bool":
		return value.Bool, true
	default:
		return nil, false
	}
}

func requiredArtifactValue[T any](value *T, err error) (any, error) {
	if value == nil {
		return nil, err
	}
	return *value, nil
}

func decodeArtifactAnyList(items list.List[ArtifactValue]) ([]any, error) {
	out := make([]any, 0, items.Len())
	for index := range items.Len() {
		item, _ := items.Get(index)
		value, err := item.Value()
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

func decodeArtifactAnyMap(items *mapping.OrderedMap[string, ArtifactValue]) (*mapping.OrderedMap[string, any], error) {
	out := mapping.NewOrderedMap[string, any]()
	if items == nil {
		return out, nil
	}
	for _, name := range items.Keys() {
		item, _ := items.Get(name)
		value, err := item.Value()
		if err != nil {
			return nil, err
		}
		out.Set(name, value)
	}
	return out, nil
}
