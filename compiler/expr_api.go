package compiler

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
)

type ExprFunctionSpec struct {
	Name  string
	Fn    func(params ...any) (any, error)
	Types list.List[any]
	Docs  string
}

func (c *Compiler) ExprVars() *mapping.OrderedMap[string, any] {
	if c == nil || c.exprVars == nil {
		return mapping.NewOrderedMap[string, any]()
	}
	return c.exprVars.Clone()
}

func (c *Compiler) ExprFunctionSpec(name string) (ExprFunctionSpec, bool) {
	if c == nil || c.exprFuncs == nil {
		return ExprFunctionSpec{}, false
	}
	return c.exprFuncs.Get(name)
}

func (c *Compiler) ExprFunctionSpecs() *mapping.OrderedMap[string, ExprFunctionSpec] {
	if c == nil || c.exprFuncs == nil {
		return mapping.NewOrderedMap[string, ExprFunctionSpec]()
	}
	return c.exprFuncs.Clone()
}

func (c *Compiler) RegisterExprVar(name string, value any) error {
	if name == "" {
		return errors.New("expr variable name cannot be empty")
	}
	c.exprVars.Set(name, value)
	c.clearExprCache()
	return nil
}

func (c *Compiler) RegisterExprFunc(name string, fn func(params ...any) (any, error), types ...any) error {
	return c.RegisterExprFunction(ExprFunctionSpec{
		Name:  name,
		Fn:    fn,
		Types: *list.NewList(types...),
	})
}

func (c *Compiler) RegisterExprFunction(spec ExprFunctionSpec) error {
	if spec.Name == "" {
		return errors.New("expr function name cannot be empty")
	}
	if spec.Fn == nil {
		return fmt.Errorf("expr function %q has nil evaluator", spec.Name)
	}
	c.exprFuncs.Set(spec.Name, spec)
	c.exprFuncSignature = ""
	c.clearExprCache()
	return nil
}
