package compiler

import (
	"reflect"
	"slices"
	"strings"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/expr-lang/expr/vm"
	lru "github.com/hashicorp/golang-lru/v2"
)

const defaultExprCacheEntries = 128

type exprCache struct {
	entries *lru.Cache[string, *vm.Program]
}

func newExprCache(entries int) *exprCache {
	if entries <= 0 {
		return nil
	}
	cache, err := lru.New[string, *vm.Program](entries)
	if err != nil {
		return nil
	}
	return &exprCache{entries: cache}
}

func normalizeExprCacheEntries(entries int) int {
	switch {
	case entries < 0:
		return 0
	case entries == 0:
		return defaultExprCacheEntries
	default:
		return entries
	}
}

func (c *exprCache) Clear() {
	if c == nil || c.entries == nil {
		return
	}
	c.entries.Purge()
}

func (c *exprCache) get(key string) (*vm.Program, bool) {
	if c == nil || c.entries == nil {
		return nil, false
	}
	return c.entries.Get(key)
}

func (c *exprCache) add(key string, program *vm.Program) {
	if c == nil || c.entries == nil || program == nil {
		return
	}
	c.entries.Add(key, program)
}

func (c *Compiler) clearExprCache() {
	if c == nil || c.exprCache == nil {
		return
	}
	c.exprCache.Clear()
}

func (c *Compiler) exprCacheKey(source string, env map[string]any) string {
	var builder strings.Builder
	builder.Grow(len(source) + len(env)*24 + len(c.exprFunctionSignature()))
	writeExprCacheString(&builder, source)
	writeExprCacheByte(&builder, '\x00')
	writeExprEnvSignature(&builder, env)
	writeExprCacheByte(&builder, '\x00')
	writeExprCacheString(&builder, c.exprFunctionSignature())
	return builder.String()
}

func (c *Compiler) exprFunctionSignature() string {
	if c == nil {
		return ""
	}
	if c.exprFuncSignature != "" {
		return c.exprFuncSignature
	}
	var builder strings.Builder
	writeExprFuncSignature(&builder, c.exprFuncs)
	c.exprFuncSignature = builder.String()
	return c.exprFuncSignature
}

func writeExprEnvSignature(builder *strings.Builder, env map[string]any) {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		writeExprCacheString(builder, key)
		writeExprCacheByte(builder, '=')
		writeExprCacheString(builder, exprValueType(env[key]))
		writeExprCacheByte(builder, ';')
	}
}

func writeExprFuncSignature(builder *strings.Builder, funcs *mapping.OrderedMap[string, ExprFunctionSpec]) {
	if funcs == nil {
		return
	}
	for _, name := range funcs.Keys() {
		spec, _ := funcs.Get(name)
		writeExprCacheString(builder, name)
		writeExprCacheByte(builder, '=')
		for _, typ := range spec.Types.Values() {
			writeExprCacheString(builder, exprValueType(typ))
			writeExprCacheByte(builder, ',')
		}
		writeExprCacheByte(builder, ';')
	}
}

func writeExprCacheString(builder *strings.Builder, value string) {
	if _, err := builder.WriteString(value); err != nil {
		panic(err)
	}
}

func writeExprCacheByte(builder *strings.Builder, value byte) {
	if err := builder.WriteByte(value); err != nil {
		panic(err)
	}
}

func exprValueType(value any) string {
	if value == nil {
		return "<nil>"
	}
	return reflect.TypeOf(value).String()
}
