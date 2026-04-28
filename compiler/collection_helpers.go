package compiler

import (
	"slices"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/samber/lo"
)

func sortedStringKeys[V any](items map[string]V) []string {
	keys := lo.Keys(items)
	slices.Sort(keys)
	return keys
}

func orderedAnyMap(items map[string]any) *mapping.OrderedMap[string, any] {
	ordered := mapping.NewOrderedMapWithCapacity[string, any](len(items))
	for _, key := range sortedStringKeys(items) {
		ordered.Set(key, items[key])
	}
	return ordered
}
