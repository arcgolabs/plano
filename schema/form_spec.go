package schema

import (
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/set"
)

func Fields(items ...FieldSpec) *mapping.OrderedMap[string, FieldSpec] {
	fields := mapping.NewOrderedMapWithCapacity[string, FieldSpec](len(items))
	for _, item := range items {
		fields.Set(item.Name, item)
	}
	return fields
}

func NestedForms(names ...string) *set.Set[string] {
	return set.NewSet(names...)
}
