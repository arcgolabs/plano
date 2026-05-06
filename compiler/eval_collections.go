package compiler

import (
	"reflect"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/schema"
)

func evalGet(args []any) (any, error) {
	switch current := args[0].(type) {
	case []any:
		return evalGetFromList(current, args[1:])
	case *mapping.OrderedMap[string, any]:
		return evalGetFromOrderedMap(current, args[1:])
	case map[string]any:
		return evalGetFromBuiltinMap(current, args[1:])
	default:
		return nil, compilerErrorf("get expects list or map")
	}
}

func evalSlice(args []any) (any, error) {
	items, err := asList(args[0], "slice")
	if err != nil {
		return nil, err
	}
	start, end, err := sliceBounds(args[1:], len(items))
	if err != nil {
		return nil, err
	}
	return append([]any(nil), items[start:end]...), nil
}

func evalHas(args []any) (any, error) {
	switch current := args[0].(type) {
	case []any:
		for _, item := range current {
			if reflect.DeepEqual(item, args[1]) {
				return true, nil
			}
		}
		return false, nil
	case *mapping.OrderedMap[string, any]:
		key, err := stringKey(args[1])
		if err != nil {
			return nil, compilerErrorf("has expects string key for map")
		}
		_, ok := current.Get(key)
		return ok, nil
	case map[string]any:
		key, err := stringKey(args[1])
		if err != nil {
			return nil, compilerErrorf("has expects string key for map")
		}
		_, ok := current[key]
		return ok, nil
	default:
		return nil, compilerErrorf("has expects list or map")
	}
}

func evalGetFromList(items, args []any) (any, error) {
	index, err := sliceIndexArg(args[0], "get")
	if err != nil {
		return nil, err
	}
	if index >= 0 && index < len(items) {
		return items[index], nil
	}
	if len(args) == 2 {
		return args[1], nil
	}
	return schema.Null{}, nil
}

func evalGetFromOrderedMap(items *mapping.OrderedMap[string, any], args []any) (any, error) {
	key, err := stringKey(args[0])
	if err != nil {
		return nil, compilerErrorf("get expects string key for map")
	}
	if value, ok := items.Get(key); ok {
		return value, nil
	}
	if len(args) == 2 {
		return args[1], nil
	}
	return schema.Null{}, nil
}

func evalGetFromBuiltinMap(items map[string]any, args []any) (any, error) {
	key, err := stringKey(args[0])
	if err != nil {
		return nil, compilerErrorf("get expects string key for map")
	}
	if value, ok := items[key]; ok {
		return value, nil
	}
	if len(args) == 2 {
		return args[1], nil
	}
	return schema.Null{}, nil
}

func evalAppend(args []any) (any, error) {
	items, err := asList(args[0], "append")
	if err != nil {
		return nil, err
	}
	result := make([]any, 0, len(items)+len(args)-1)
	result = append(result, items...)
	result = append(result, args[1:]...)
	return result, nil
}

func evalConcat(args []any) (any, error) {
	result := make([]any, 0)
	for _, arg := range args {
		items, err := asList(arg, "concat")
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
	}
	return result, nil
}

func evalMerge(args []any) (any, error) {
	result, err := asOrderedMap(args[0], "merge")
	if err != nil {
		return nil, err
	}
	for _, arg := range args[1:] {
		current, err := asOrderedMap(arg, "merge")
		if err != nil {
			return nil, err
		}
		current.Range(func(key string, value any) bool {
			result.Set(key, value)
			return true
		})
	}
	return result, nil
}

func asList(value any, name string) ([]any, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, compilerErrorf("%s expects list arguments", name)
	}
	return items, nil
}

func sliceBounds(args []any, size int) (int, int, error) {
	start, err := sliceIndexArg(args[0], "slice")
	if err != nil {
		return 0, 0, err
	}
	end := size
	if len(args) == 2 {
		end, err = sliceIndexArg(args[1], "slice")
		if err != nil {
			return 0, 0, err
		}
	}
	if start < 0 || end < start || end > size {
		return 0, 0, compilerErrorf("slice bounds are out of range")
	}
	return start, end, nil
}

func sliceIndexArg(value any, name string) (int, error) {
	index, ok := value.(int64)
	if !ok {
		return 0, compilerErrorf("%s expects int index arguments", name)
	}
	return int(index), nil
}

func asOrderedMap(value any, name string) (*mapping.OrderedMap[string, any], error) {
	switch current := value.(type) {
	case *mapping.OrderedMap[string, any]:
		return current.Clone(), nil
	case map[string]any:
		return orderedAnyMap(current), nil
	default:
		return nil, compilerErrorf("%s expects map arguments", name)
	}
}
