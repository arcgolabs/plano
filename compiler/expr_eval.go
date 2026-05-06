package compiler

import (
	"cmp"
	"math"
	"reflect"
	"slices"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/schema"
	exprlang "github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

func (s *compileState) evalExprLangCall(name string, args []any, locals *env) (any, bool, error) {
	if name != "expr" && name != "expr_eval" {
		return nil, false, nil
	}
	if len(args) == 0 {
		return nil, true, compilerErrorf("expr expects expression string")
	}
	source, ok := args[0].(string)
	if !ok {
		return nil, true, compilerErrorf("expr expects expression string")
	}
	env, err := s.exprLangEnv(locals, args[1:])
	if err != nil {
		return nil, true, err
	}
	options := []exprlang.Option{
		exprlang.Env(env),
		exprlang.AsAny(),
	}
	s.compiler.exprFuncs.Range(func(_ string, spec ExprFunctionSpec) bool {
		options = append(options, exprlang.Function(spec.Name, spec.Fn, spec.Types.Values()...))
		return true
	})
	cacheKey := s.compiler.exprCacheKey(source, env)
	program, err := s.compileExprLangProgram(cacheKey, source, options)
	if err != nil {
		return nil, true, err
	}
	value, err := exprlang.Run(program, env)
	if err != nil {
		return nil, true, wrapCompilerErrorf(err, "run expr expression")
	}
	return normalizeExprLangValue(value), true, nil
}

func (s *compileState) compileExprLangProgram(
	cacheKey string,
	source string,
	options []exprlang.Option,
) (*vm.Program, error) {
	if program, ok := s.compiler.exprCache.get(cacheKey); ok {
		return program, nil
	}
	program, err := exprlang.Compile(source, options...)
	if err != nil {
		return nil, wrapCompilerErrorf(err, "compile expr expression")
	}
	s.compiler.exprCache.add(cacheKey, program)
	return program, nil
}

func (s *compileState) exprLangEnv(locals *env, overrides []any) (map[string]any, error) {
	env := make(map[string]any)
	s.compiler.globals.Range(func(name string, value any) bool {
		env[name] = exprLangValue(value)
		return true
	})
	s.compiler.exprVars.Range(func(name string, value any) bool {
		env[name] = exprLangValue(value)
		return true
	})
	s.constValues.Range(func(name string, value any) bool {
		env[name] = exprLangValue(value)
		return true
	})
	if locals != nil {
		locals.exprLangValues().Range(func(name string, value any) bool {
			env[name] = exprLangValue(value)
			return true
		})
	}
	if len(overrides) == 0 {
		return env, nil
	}
	overrideMap, err := exprLangOverrideMap(overrides[0])
	if err != nil {
		return nil, err
	}
	overrideMap.Range(func(name string, value any) bool {
		env[name] = exprLangValue(value)
		return true
	})
	return env, nil
}

func (e *env) exprLangValues() *mapping.OrderedMap[string, any] {
	out := mapping.NewOrderedMap[string, any]()
	frames := make([]*env, 0)
	for current := e; current != nil; current = current.parent {
		frames = append(frames, current)
	}
	for index := len(frames) - 1; index >= 0; index-- {
		frame := frames[index]
		frame.values.Range(func(name string, binding envBinding) bool {
			out.Set(name, binding.value)
			return true
		})
	}
	return out
}

func exprLangOverrideMap(value any) (*mapping.OrderedMap[string, any], error) {
	switch current := value.(type) {
	case *mapping.OrderedMap[string, any]:
		return current.Clone(), nil
	case map[string]any:
		return orderedAnyMap(current), nil
	default:
		return nil, compilerErrorf("expr override expects map, got %T", value)
	}
}

func exprLangValue(value any) any {
	switch current := value.(type) {
	case schema.Ref:
		return map[string]any{
			"kind": current.Kind,
			"name": current.Name,
		}
	case *mapping.OrderedMap[string, any]:
		out := make(map[string]any, current.Len())
		current.Range(func(key string, item any) bool {
			out[key] = exprLangValue(item)
			return true
		})
		return out
	case map[string]any:
		out := make(map[string]any, len(current))
		for key, item := range current {
			out[key] = exprLangValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(current))
		for _, item := range current {
			out = append(out, exprLangValue(item))
		}
		return out
	default:
		return value
	}
}

func normalizeExprLangValue(value any) any {
	if normalized, ok := normalizeExprScalar(value); ok {
		return normalized
	}
	if normalized, ok := normalizeExprCollection(value); ok {
		return normalized
	}
	return value
}

func normalizeExprScalar(value any) (any, bool) {
	if normalized, ok := normalizeExprSignedInt(value); ok {
		return normalized, true
	}
	if normalized, ok := normalizeExprUnsignedInt(value); ok {
		return normalized, true
	}
	switch current := value.(type) {
	case float32:
		return float64(current), true
	case string, bool, nil:
		return current, true
	default:
		return nil, false
	}
}

func normalizeExprSignedInt(value any) (any, bool) {
	switch current := value.(type) {
	case int:
		return int64(current), true
	case int8:
		return int64(current), true
	case int16:
		return int64(current), true
	case int32:
		return int64(current), true
	case int64:
		return current, true
	default:
		return nil, false
	}
}

func normalizeExprUnsignedInt(value any) (any, bool) {
	switch current := value.(type) {
	case uint:
		return uintToInt64(uint64(current)), true
	case uint8:
		return int64(current), true
	case uint16:
		return int64(current), true
	case uint32:
		return int64(current), true
	case uint64:
		return uintToInt64(current), true
	default:
		return nil, false
	}
}

func normalizeExprCollection(value any) (any, bool) {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil, true
	}
	kind := rv.Kind()
	if kind == reflect.Slice || kind == reflect.Array {
		return normalizeExprSlice(rv), true
	}
	if kind == reflect.Map {
		return normalizeExprMap(rv)
	}
	return nil, false
}

func normalizeExprSlice(rv reflect.Value) []any {
	out := make([]any, 0, rv.Len())
	for index := range rv.Len() {
		out = append(out, normalizeExprLangValue(rv.Index(index).Interface()))
	}
	return out
}

func normalizeExprMap(rv reflect.Value) (any, bool) {
	if rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	out := mapping.NewOrderedMapWithCapacity[string, any](rv.Len())
	keys := rv.MapKeys()
	slices.SortFunc(keys, func(left, right reflect.Value) int {
		return cmp.Compare(left.String(), right.String())
	})
	for _, key := range keys {
		out.Set(key.String(), normalizeExprLangValue(rv.MapIndex(key).Interface()))
	}
	return out, true
}

func uintToInt64(value uint64) any {
	if value > math.MaxInt64 {
		return float64(value)
	}
	return int64(value)
}
