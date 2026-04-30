package compiler

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/lo"
)

func (c *Compiler) registerBuiltins() {
	c.mustRegisterFunc(builtinFunction("env", "Read an environment variable with an optional fallback string.", 1, 2, schema.Types(schema.TypeString, schema.TypeString), nil, schema.TypeString, func(args []any) (any, error) {
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
	c.mustRegisterFunc(builtinFunction("join_path", "Join one or more path fragments into a normalized path.", 1, -1, schema.Types(schema.TypePath), schema.TypePath, schema.TypePath, evalJoinPath))
	c.mustRegisterFunc(builtinFunction("basename", "Return the last path element.", 1, 1, schema.Types(schema.TypePath), nil, schema.TypeString, evalBaseName))
	c.mustRegisterFunc(builtinFunction("dirname", "Return the parent directory of a path.", 1, 1, schema.Types(schema.TypePath), nil, schema.TypePath, evalDirName))
	c.mustRegisterFunc(builtinFunction("len", "Return the length of a string, list, or map.", 1, 1, schema.Types(schema.TypeAny), nil, schema.TypeInt, evalLen))
	c.mustRegisterFunc(builtinFunction("keys", "Return the ordered string keys of a map value.", 1, 1, schema.Types(schema.MapType{Elem: schema.TypeAny}), nil, schema.ListType{Elem: schema.TypeString}, evalKeys))
	c.mustRegisterFunc(builtinFunction("values", "Return the ordered values of a map value.", 1, 1, schema.Types(schema.MapType{Elem: schema.TypeAny}), nil, schema.ListType{Elem: schema.TypeAny}, evalValues))
	c.mustRegisterFunc(builtinFunction("range", "Build an integer list from start, end, and optional step values.", 1, 3, schema.Types(schema.TypeInt, schema.TypeInt, schema.TypeInt), nil, schema.ListType{Elem: schema.TypeInt}, evalRange))
	c.mustRegisterFunc(builtinFunction("get", "Read a list or map element with an optional default value.", 2, 3, schema.Types(schema.TypeAny, schema.TypeAny, schema.TypeAny), nil, schema.TypeAny, evalGet))
	c.mustRegisterFunc(builtinFunction("slice", "Return a sub-slice from a list using start and optional end indexes.", 2, 3, schema.Types(schema.ListType{Elem: schema.TypeAny}, schema.TypeInt, schema.TypeInt), nil, schema.ListType{Elem: schema.TypeAny}, evalSlice))
	c.mustRegisterFunc(builtinFunction("has", "Report whether a list contains a value or a map contains a key.", 2, 2, schema.Types(schema.TypeAny, schema.TypeAny), nil, schema.TypeBool, evalHas))
	c.mustRegisterFunc(builtinFunction("append", "Append one or more values to a list.", 1, -1, schema.Types(schema.ListType{Elem: schema.TypeAny}), schema.TypeAny, schema.ListType{Elem: schema.TypeAny}, evalAppend))
	c.mustRegisterFunc(builtinFunction("concat", "Concatenate multiple lists into one list.", 1, -1, schema.Types(schema.ListType{Elem: schema.TypeAny}), schema.ListType{Elem: schema.TypeAny}, schema.ListType{Elem: schema.TypeAny}, evalConcat))
	c.mustRegisterFunc(builtinFunction("merge", "Merge one or more maps from left to right.", 1, -1, schema.Types(schema.MapType{Elem: schema.TypeAny}), schema.MapType{Elem: schema.TypeAny}, schema.MapType{Elem: schema.TypeAny}, evalMerge))
	c.mustRegisterFunc(builtinFunction("expr", "Evaluate an expr-lang expression string with the current plano evaluation environment.", 1, 2, schema.Types(schema.TypeString, schema.MapType{Elem: schema.TypeAny}), nil, schema.TypeAny, evalExprPlaceholder))
	c.mustRegisterFunc(builtinFunction("expr_eval", "Evaluate an expr-lang expression string with the current plano evaluation environment.", 1, 2, schema.Types(schema.TypeString, schema.MapType{Elem: schema.TypeAny}), nil, schema.TypeAny, evalExprPlaceholder))
}

func (c *Compiler) mustRegisterFunc(spec schema.FunctionSpec) {
	if err := c.RegisterFunc(spec); err != nil {
		panic(fmt.Errorf("register builtin function %q: %w", spec.Name, err))
	}
}

func builtinFunction(
	name string,
	docs string,
	minArgs int,
	maxArgs int,
	paramTypes list.List[schema.Type],
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
		Eval: func(args list.List[any]) (any, error) {
			return eval(args.Values())
		},
		Docs: docs,
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
		return stringSliceAsAny(sortedStringKeys(current)), nil
	default:
		return nil, errors.New("keys expects map argument")
	}
}

func evalValues(args []any) (any, error) {
	switch current := args[0].(type) {
	case *mapping.OrderedMap[string, any]:
		return slices.Clone(current.Values()), nil
	case map[string]any:
		return slices.Clone(orderedAnyMap(current).Values()), nil
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
	return lo.Map(items, func(item string, _ int) any {
		return item
	})
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

func evalExprPlaceholder([]any) (any, error) {
	return nil, errors.New("expr evaluation requires compiler runtime context")
}
