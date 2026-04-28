package compiler

import (
	"fmt"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/lo"
)

func staticTypeOfValue(value any) schema.Type {
	if typ, ok := scalarTypeOfValue(value); ok {
		return typ
	}
	if typ, ok := collectionTypeOfValue(value); ok {
		return typ
	}
	return schema.TypeAny
}

func typesOfValues(values []any) []schema.Type {
	return lo.Map(values, func(value any, _ int) schema.Type {
		return staticTypeOfValue(value)
	})
}

func typesOfMapValues(values *mapping.OrderedMap[string, any]) []schema.Type {
	return lo.Map(values.Values(), func(value any, _ int) schema.Type {
		return staticTypeOfValue(value)
	})
}

func mergeTypes(types []schema.Type) schema.Type {
	if len(types) == 0 {
		return schema.TypeAny
	}
	current := normalizeType(types[0])
	for _, item := range types[1:] {
		item = normalizeType(item)
		if !isTypeAssignable(current, item) || !isTypeAssignable(item, current) {
			return schema.TypeAny
		}
	}
	return current
}

func normalizeType(typ schema.Type) schema.Type {
	if typ == nil {
		return schema.TypeAny
	}
	return typ
}

func isTypeAssignable(want, actual schema.Type) bool {
	want = normalizeType(want)
	actual = normalizeType(actual)

	if want == schema.TypeAny || actual == schema.TypeAny {
		return true
	}
	if ok, matched := builtinAssignable(want, actual); matched {
		return ok
	}
	if ok, matched := containerAssignable(want, actual); matched {
		return ok
	}
	if ok, matched := refAssignable(want, actual); matched {
		return ok
	}
	if ok, matched := namedAssignable(want, actual); matched {
		return ok
	}
	return false
}

func scalarTypeOfValue(value any) (schema.Type, bool) {
	switch current := value.(type) {
	case nil, schema.Null:
		return schema.TypeAny, true
	case string:
		return schema.TypeString, true
	case int64:
		return schema.TypeInt, true
	case float64:
		return schema.TypeFloat, true
	case bool:
		return schema.TypeBool, true
	case schema.Duration:
		return schema.TypeDuration, true
	case schema.Size:
		return schema.TypeSize, true
	case schema.Ref:
		return schema.RefType{Kind: current.Kind}, true
	default:
		return nil, false
	}
}

func collectionTypeOfValue(value any) (schema.Type, bool) {
	switch current := value.(type) {
	case []any:
		if len(current) == 0 {
			return schema.ListType{Elem: schema.TypeAny}, true
		}
		return schema.ListType{Elem: mergeTypes(typesOfValues(current))}, true
	case *mapping.OrderedMap[string, any]:
		if current.Len() == 0 {
			return schema.MapType{Elem: schema.TypeAny}, true
		}
		return schema.MapType{Elem: mergeTypes(typesOfMapValues(current))}, true
	case map[string]any:
		if len(current) == 0 {
			return schema.MapType{Elem: schema.TypeAny}, true
		}
		return schema.MapType{Elem: mergeTypes(typesOfMapValues(orderedAnyMap(current)))}, true
	default:
		return nil, false
	}
}

func builtinAssignable(want, actual schema.Type) (bool, bool) {
	expected, ok := want.(schema.BuiltinType)
	if !ok {
		return false, false
	}
	actualBuiltin, ok := actual.(schema.BuiltinType)
	if !ok {
		return false, true
	}
	return expected == actualBuiltin || isStringCompatible(expected, actualBuiltin), true
}

func containerAssignable(want, actual schema.Type) (bool, bool) {
	switch expected := want.(type) {
	case schema.ListType:
		actualList, ok := actual.(schema.ListType)
		return ok && isTypeAssignable(expected.Elem, actualList.Elem), true
	case schema.MapType:
		actualMap, ok := actual.(schema.MapType)
		return ok && isTypeAssignable(expected.Elem, actualMap.Elem), true
	default:
		return false, false
	}
}

func refAssignable(want, actual schema.Type) (bool, bool) {
	expected, ok := want.(schema.RefType)
	if !ok {
		return false, false
	}
	actualRef, ok := actual.(schema.RefType)
	if !ok {
		return false, true
	}
	return expected.Kind == actualRef.Kind, true
}

func namedAssignable(want, actual schema.Type) (bool, bool) {
	expected, ok := want.(schema.NamedType)
	if !ok {
		return false, false
	}
	actualNamed, ok := actual.(schema.NamedType)
	if !ok {
		return false, true
	}
	return expected.Name == actualNamed.Name, true
}

func isStringCompatible(left, right schema.BuiltinType) bool {
	if left == schema.TypeString && right == schema.TypePath {
		return true
	}
	return left == schema.TypePath && right == schema.TypeString
}

func typeMismatchError(label string, want, actual schema.Type) error {
	return fmt.Errorf("%s expects %s, got %s", label, normalizeType(want).String(), normalizeType(actual).String())
}

func isStringType(typ schema.Type) bool {
	builtin, ok := normalizeType(typ).(schema.BuiltinType)
	return ok && (builtin == schema.TypeString || builtin == schema.TypePath)
}

func isNumericType(typ schema.Type) bool {
	builtin, ok := normalizeType(typ).(schema.BuiltinType)
	return ok && (builtin == schema.TypeInt || builtin == schema.TypeFloat)
}
