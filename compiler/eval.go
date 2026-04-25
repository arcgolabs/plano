package compiler

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/arcgolabs/plano/schema"
)

func (c *Compiler) registerBuiltins() {
	c.mustRegisterFunc(builtinFunction("env", 2, []schema.Type{schema.TypeString, schema.TypeString}, nil, schema.TypeString, func(args []any) (any, error) {
		key, ok := args[0].(string)
		if !ok {
			return nil, errors.New("env expects string key")
		}
		if value, ok := c.lookupEnv(key); ok {
			return value, nil
		}
		if len(args) == 2 {
			if fallback, ok := args[1].(string); ok {
				return fallback, nil
			}
			return nil, errors.New("env fallback expects string")
		}
		return "", nil
	}))
	c.mustRegisterFunc(builtinFunction("join_path", -1, []schema.Type{schema.TypePath}, schema.TypePath, schema.TypePath, evalJoinPath))
	c.mustRegisterFunc(builtinFunction("basename", 1, []schema.Type{schema.TypePath}, nil, schema.TypeString, evalBaseName))
	c.mustRegisterFunc(builtinFunction("dirname", 1, []schema.Type{schema.TypePath}, nil, schema.TypePath, evalDirName))
}

func (c *Compiler) mustRegisterFunc(spec schema.FunctionSpec) {
	if err := c.RegisterFunc(spec); err != nil {
		panic(fmt.Errorf("register builtin function %q: %w", spec.Name, err))
	}
}

func builtinFunction(
	name string,
	maxArgs int,
	paramTypes []schema.Type,
	variadicType schema.Type,
	result schema.Type,
	eval func(args []any) (any, error),
) schema.FunctionSpec {
	return schema.FunctionSpec{
		Name:         name,
		MinArgs:      1,
		MaxArgs:      maxArgs,
		ParamTypes:   paramTypes,
		VariadicType: variadicType,
		Result:       result,
		Eval:         eval,
	}
}

func evalJoinPath(args []any) (any, error) {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		text, ok := arg.(string)
		if !ok {
			return nil, errors.New("join_path expects string arguments")
		}
		parts = append(parts, text)
	}
	return filepath.Join(parts...), nil
}

func evalBaseName(args []any) (any, error) {
	path, ok := args[0].(string)
	if !ok {
		return nil, errors.New("basename expects string argument")
	}
	return filepath.Base(path), nil
}

func evalDirName(args []any) (any, error) {
	path, ok := args[0].(string)
	if !ok {
		return nil, errors.New("dirname expects string argument")
	}
	return filepath.Dir(path), nil
}
