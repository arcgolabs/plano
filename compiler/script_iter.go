package compiler

import (
	"fmt"

	"github.com/arcgolabs/collectionx/mapping"
)

type iterItem struct {
	Key   any
	Value any
}

func iterateItems(value any) ([]iterItem, error) {
	switch current := value.(type) {
	case []any:
		items := make([]iterItem, 0, len(current))
		for idx, item := range current {
			items = append(items, iterItem{
				Key:   int64(idx),
				Value: item,
			})
		}
		return items, nil
	case *mapping.OrderedMap[string, any]:
		items := make([]iterItem, 0, current.Len())
		current.Range(func(key string, value any) bool {
			items = append(items, iterItem{
				Key:   key,
				Value: value,
			})
			return true
		})
		return items, nil
	case map[string]any:
		keys := sortedStringKeys(current)
		items := make([]iterItem, 0, len(keys))
		for _, key := range keys {
			items = append(items, iterItem{
				Key:   key,
				Value: current[key],
			})
		}
		return items, nil
	default:
		return nil, fmt.Errorf("for loop expects list or map, got %T", value)
	}
}
