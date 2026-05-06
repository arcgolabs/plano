package compiler

import (
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/schema"
)

type envBinding struct {
	kind  LocalBindingKind
	typ   schema.Type
	value any
}

type env struct {
	parent *env
	scope  string
	values *mapping.OrderedMap[string, envBinding]
}

func newEnv(parent *env, scope string) *env {
	return &env{
		parent: parent,
		scope:  scope,
		values: mapping.NewOrderedMap[string, envBinding](),
	}
}

func (e *env) BindLocal(name string, kind LocalBindingKind, typ schema.Type, value any) {
	e.values.Set(name, envBinding{
		kind:  kind,
		typ:   normalizeType(typ),
		value: value,
	})
}

func (e *env) Get(name string) (any, bool) {
	for current := e; current != nil; current = current.parent {
		if binding, ok := current.values.Get(name); ok {
			return binding.value, true
		}
	}
	return nil, false
}

func (e *env) Lookup(name string) (envBinding, bool) {
	for current := e; current != nil; current = current.parent {
		if binding, ok := current.values.Get(name); ok {
			return binding, true
		}
	}
	return envBinding{}, false
}

func (e *env) Assign(name string, value any) error {
	for current := e; current != nil; current = current.parent {
		binding, ok := current.values.Get(name)
		if !ok {
			continue
		}
		if binding.kind == LocalConst {
			return compilerErrorf("cannot assign to const %q", name)
		}
		if err := schema.CheckAssignable(binding.typ, value); err != nil {
			return wrapCompilerErrorf(err, "assignment %q", name)
		}
		binding.value = value
		current.values.Set(name, binding)
		return nil
	}
	return compilerErrorf("undefined local binding %q", name)
}
