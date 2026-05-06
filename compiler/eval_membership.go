package compiler

import (
	"reflect"

	"github.com/arcgolabs/collectionx/mapping"
)

func evalContains(collection, item any) (bool, error) {
	switch current := collection.(type) {
	case []any:
		return evalListContains(current, item), nil
	case *mapping.OrderedMap[string, any]:
		return evalOrderedMapContains(current, item)
	case map[string]any:
		return evalBuiltinMapContains(current, item)
	default:
		return false, compilerErrorf("operator in expects list or map")
	}
}

func evalListContains(items []any, item any) bool {
	for _, candidate := range items {
		if reflect.DeepEqual(candidate, item) {
			return true
		}
	}
	return false
}

func evalOrderedMapContains(items *mapping.OrderedMap[string, any], item any) (bool, error) {
	key, err := stringKey(item)
	if err != nil {
		return false, compilerErrorf("operator in expects string key for map")
	}
	_, ok := items.Get(key)
	return ok, nil
}

func evalBuiltinMapContains(items map[string]any, item any) (bool, error) {
	key, err := stringKey(item)
	if err != nil {
		return false, compilerErrorf("operator in expects string key for map")
	}
	_, ok := items[key]
	return ok, nil
}
