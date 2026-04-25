package compiler

import "github.com/arcgolabs/collectionx/mapping"

type env struct {
	parent *env
	scope  string
	values *mapping.OrderedMap[string, any]
}

func newEnv(parent *env, scope string) *env {
	return &env{
		parent: parent,
		scope:  scope,
		values: mapping.NewOrderedMap[string, any](),
	}
}

func (e *env) Bind(name string, value any) {
	e.values.Set(name, value)
}

func (e *env) Get(name string) (any, bool) {
	for current := e; current != nil; current = current.parent {
		if value, ok := current.values.Get(name); ok {
			return value, true
		}
	}
	return nil, false
}
