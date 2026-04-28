package schema

import (
	"github.com/arcgolabs/collectionx/list"
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

func Types(items ...Type) list.List[Type] {
	if len(items) == 0 {
		return list.List[Type]{}
	}
	return *list.NewList(items...)
}

func FormSpecs(items ...FormSpec) list.List[FormSpec] {
	if len(items) == 0 {
		return list.List[FormSpec]{}
	}
	return *list.NewList(items...)
}
