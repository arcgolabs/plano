package compiler

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/schema"
)

func (c *Compiler) registerBuiltins() {
	c.mustRegisterFunc(builtinFunction("env", 1, 2, []schema.Type{schema.TypeString, schema.TypeString}, nil, schema.TypeString, func(args []any) (any, error) {
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
	c.mustRegisterFunc(builtinFunction("join_path", 1, -1, []schema.Type{schema.TypePath}, schema.TypePath, schema.TypePath, evalJoinPath))
	c.mustRegisterFunc(builtinFunction("basename", 1, 1, []schema.Type{schema.TypePath}, nil, schema.TypeString, evalBaseName))
	c.mustRegisterFunc(builtinFunction("dirname", 1, 1, []schema.Type{schema.TypePath}, nil, schema.TypePath, evalDirName))
	c.mustRegisterFunc(builtinFunction("len", 1, 1, []schema.Type{schema.TypeAny}, nil, schema.TypeInt, evalLen))
	c.mustRegisterFunc(builtinFunction("keys", 1, 1, []schema.Type{schema.MapType{Elem: schema.TypeAny}}, nil, schema.ListType{Elem: schema.TypeString}, evalKeys))
	c.mustRegisterFunc(builtinFunction("values", 1, 1, []schema.Type{schema.MapType{Elem: schema.TypeAny}}, nil, schema.ListType{Elem: schema.TypeAny}, evalValues))
	c.mustRegisterFunc(builtinFunction("range", 1, 3, []schema.Type{schema.TypeInt, schema.TypeInt, schema.TypeInt}, nil, schema.ListType{Elem: schema.TypeInt}, evalRange))
	c.mustRegisterFunc(builtinFunction("get", 2, 3, []schema.Type{schema.TypeAny, schema.TypeAny, schema.TypeAny}, nil, schema.TypeAny, evalGet))
	c.mustRegisterFunc(builtinFunction("slice", 2, 3, []schema.Type{schema.ListType{Elem: schema.TypeAny}, schema.TypeInt, schema.TypeInt}, nil, schema.ListType{Elem: schema.TypeAny}, evalSlice))
	c.mustRegisterFunc(builtinFunction("has", 2, 2, []schema.Type{schema.TypeAny, schema.TypeAny}, nil, schema.TypeBool, evalHas))
	c.mustRegisterFunc(builtinFunction("append", 1, -1, []schema.Type{schema.ListType{Elem: schema.TypeAny}}, schema.TypeAny, schema.ListType{Elem: schema.TypeAny}, evalAppend))
	c.mustRegisterFunc(builtinFunction("concat", 1, -1, []schema.Type{schema.ListType{Elem: schema.TypeAny}}, schema.ListType{Elem: schema.TypeAny}, schema.ListType{Elem: schema.TypeAny}, evalConcat))
	c.mustRegisterFunc(builtinFunction("merge", 1, -1, []schema.Type{schema.MapType{Elem: schema.TypeAny}}, schema.MapType{Elem: schema.TypeAny}, schema.MapType{Elem: schema.TypeAny}, evalMerge))
}

func (c *Compiler) mustRegisterFunc(spec schema.FunctionSpec) {
	if err := c.RegisterFunc(spec); err != nil {
		panic(fmt.Errorf("register builtin function %q: %w", spec.Name, err))
	}
}

func builtinFunction(
	name string,
	minArgs int,
	maxArgs int,
	paramTypes []schema.Type,
	variadicType schema.Type,
	result schema.Type,
	eval func(args []any) (any, error),
) schema.FunctionSpec {
	return schema.FunctionSpec{
		Name:         name,
		MinArgs:      minArgs,
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

func evalLen(args []any) (any, error) {
	switch current := args[0].(type) {
	case string:
		return int64(len(current)), nil
	case []any:
		return int64(len(current)), nil
	case *mapping.OrderedMap[string, any]:
		return int64(current.Len()), nil
	case map[string]any:
		return int64(len(current)), nil
	default:
		return nil, errors.New("len expects string, list, or map")
	}
}

func evalKeys(args []any) (any, error) {
	switch current := args[0].(type) {
	case *mapping.OrderedMap[string, any]:
		return stringSliceAsAny(current.Keys()), nil
	case map[string]any:
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		return stringSliceAsAny(keys), nil
	default:
		return nil, errors.New("keys expects map argument")
	}
}

func evalValues(args []any) (any, error) {
	switch current := args[0].(type) {
	case *mapping.OrderedMap[string, any]:
		return slices.Clone(current.Values()), nil
	case map[string]any:
		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		values := make([]any, 0, len(keys))
		for _, key := range keys {
			values = append(values, current[key])
		}
		return values, nil
	default:
		return nil, errors.New("values expects map argument")
	}
}

func evalRange(args []any) (any, error) {
	start, end, step, err := parseRangeArgs(args)
	if err != nil {
		return nil, err
	}
	if step == 0 {
		return nil, errors.New("range step must not be zero")
	}
	return buildRange(start, end, step), nil
}

func stringSliceAsAny(items []string) []any {
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, item)
	}
	return values
}

func parseRangeArgs(args []any) (int64, int64, int64, error) {
	if len(args) == 0 {
		return 0, 0, 0, errors.New("range expects int arguments")
	}
	end, err := rangeIntArg(args[0])
	if err != nil {
		return 0, 0, 0, err
	}
	start := int64(0)
	step := int64(1)
	if len(args) >= 2 {
		start, err = rangeIntArg(args[0])
		if err != nil {
			return 0, 0, 0, err
		}
		end, err = rangeIntArg(args[1])
		if err != nil {
			return 0, 0, 0, err
		}
	}
	if len(args) == 3 {
		step, err = rangeIntArg(args[2])
		if err != nil {
			return 0, 0, 0, err
		}
	}
	return start, end, step, nil
}

func rangeIntArg(value any) (int64, error) {
	result, ok := value.(int64)
	if !ok {
		return 0, errors.New("range expects int arguments")
	}
	return result, nil
}

func buildRange(start, end, step int64) []any {
	values := make([]any, 0)
	if step > 0 {
		for value := start; value < end; value += step {
			values = append(values, value)
		}
		return values
	}
	for value := start; value > end; value += step {
		values = append(values, value)
	}
	return values
}
