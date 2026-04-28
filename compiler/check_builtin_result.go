package compiler

import "github.com/arcgolabs/plano/schema"

func getResultType(argTypes []schema.Type, fallback schema.Type) schema.Type {
	if len(argTypes) == 0 {
		return normalizeType(fallback)
	}
	base := normalizeType(argTypes[0])
	var result schema.Type
	switch current := base.(type) {
	case schema.ListType:
		result = normalizeType(current.Elem)
	case schema.MapType:
		result = normalizeType(current.Elem)
	default:
		return normalizeType(fallback)
	}
	if len(argTypes) == 3 {
		return mergeTypes([]schema.Type{result, normalizeType(argTypes[2])})
	}
	return schema.TypeAny
}

func sliceResultType(argTypes []schema.Type, fallback schema.Type) schema.Type {
	if len(argTypes) == 0 {
		return normalizeType(fallback)
	}
	if current, ok := normalizeType(argTypes[0]).(schema.ListType); ok {
		return schema.ListType{Elem: normalizeType(current.Elem)}
	}
	return normalizeType(fallback)
}

func valuesResultType(argTypes []schema.Type, fallback schema.Type) schema.Type {
	if len(argTypes) == 0 {
		return normalizeType(fallback)
	}
	current, ok := normalizeType(argTypes[0]).(schema.MapType)
	if !ok {
		return normalizeType(fallback)
	}
	return schema.ListType{Elem: normalizeType(current.Elem)}
}

func appendResultType(argTypes []schema.Type, fallback schema.Type) schema.Type {
	if len(argTypes) == 0 {
		return normalizeType(fallback)
	}
	elem := listElemType(argTypes[0])
	if len(argTypes) == 1 {
		return schema.ListType{Elem: elem}
	}
	types := make([]schema.Type, 0, len(argTypes))
	types = append(types, elem)
	types = append(types, argTypes[1:]...)
	return schema.ListType{Elem: mergeTypes(types)}
}

func concatResultType(argTypes []schema.Type, fallback schema.Type) schema.Type {
	if len(argTypes) == 0 {
		return normalizeType(fallback)
	}
	types := make([]schema.Type, 0, len(argTypes))
	for _, argType := range argTypes {
		types = append(types, listElemType(argType))
	}
	return schema.ListType{Elem: mergeTypes(types)}
}

func mergeResultType(argTypes []schema.Type, fallback schema.Type) schema.Type {
	if len(argTypes) == 0 {
		return normalizeType(fallback)
	}
	types := make([]schema.Type, 0, len(argTypes))
	for _, argType := range argTypes {
		types = append(types, mapElemType(argType))
	}
	return schema.MapType{Elem: mergeTypes(types)}
}

func listElemType(typ schema.Type) schema.Type {
	current, ok := normalizeType(typ).(schema.ListType)
	if !ok {
		return schema.TypeAny
	}
	return normalizeType(current.Elem)
}

func mapElemType(typ schema.Type) schema.Type {
	current, ok := normalizeType(typ).(schema.MapType)
	if !ok {
		return schema.TypeAny
	}
	return normalizeType(current.Elem)
}
