package compiler

import (
	"context"
	"go/token"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

type checkScope struct {
	id     string
	kind   ScopeKind
	parent *checkScope
	locals *mapping.Map[string, checkLocalBinding]
}

type checkLocalBinding struct {
	kind LocalBindingKind
	typ  schema.Type
}

type checkScopeKey struct {
	kind     ScopeKind
	parentID string
	pos      token.Pos
	end      token.Pos
}

type checker struct {
	compiler    *Compiler
	binding     *Binding
	constDecls  *mapping.OrderedMap[string, *ast.ConstDecl]
	funcDecls   *mapping.OrderedMap[string, *ast.FnDecl]
	scopeIndex  *mapping.Map[checkScopeKey, string]
	checks      *CheckInfo
	constTypes  *mapping.Map[string, schema.Type]
	resolving   *set.Set[string]
	nextExpr    int
	nextField   int
	nextCall    int
	diagnostics diag.Diagnostics
}

func (c *Compiler) CheckSource(ctx context.Context, filename string, src []byte) (*CheckInfo, diag.Diagnostics) {
	result := c.CheckSourceDetailed(ctx, filename, src)
	return result.Checks, result.Diagnostics
}

func (c *Compiler) CheckFile(ctx context.Context, filename string) (*CheckInfo, diag.Diagnostics) {
	result := c.CheckFileDetailed(ctx, filename)
	return result.Checks, result.Diagnostics
}

func (c *Compiler) CheckSourceDetailed(ctx context.Context, filename string, src []byte) CheckResult {
	_ = ctx
	input := c.prepareSource(filename, src)
	index := c.bindUnits(input.units)
	input.diagnostics.Append(index.diagnostics)
	checks, more := c.checkUnits(input.units, index)
	input.diagnostics.Append(more)
	return CheckResult{
		Binding:     index.binding,
		Checks:      checks,
		FileSet:     input.fileSet,
		Diagnostics: input.diagnostics,
	}
}

func (c *Compiler) CheckFileDetailed(ctx context.Context, filename string) CheckResult {
	_ = ctx
	input := c.prepareFile(filename)
	index := c.bindUnits(input.units)
	input.diagnostics.Append(index.diagnostics)
	checks, more := c.checkUnits(input.units, index)
	input.diagnostics.Append(more)
	return CheckResult{
		Binding:     index.binding,
		Checks:      checks,
		FileSet:     input.fileSet,
		Diagnostics: input.diagnostics,
	}
}

func (c *Compiler) checkUnits(units []parsedUnit, index boundIndex) (*CheckInfo, diag.Diagnostics) {
	checks := &CheckInfo{
		Exprs:  mapping.NewOrderedMap[string, ExprCheck](),
		Fields: mapping.NewOrderedMap[string, FieldCheck](),
		Calls:  mapping.NewOrderedMap[string, CallCheck](),
	}
	k := checker{
		compiler:   c,
		binding:    index.binding,
		constDecls: index.constDecls,
		funcDecls:  index.funcDecls,
		scopeIndex: buildCheckScopeIndex(index.binding),
		checks:     checks,
		constTypes: mapping.NewMap[string, schema.Type](),
		resolving:  set.NewSet[string](),
	}
	k.checkAllConsts()
	k.checkAllUnits(units)
	return checks, k.diagnostics
}

func buildCheckScopeIndex(binding *Binding) *mapping.Map[checkScopeKey, string] {
	index := mapping.NewMapWithCapacity[checkScopeKey, string](binding.Scopes.Len())
	binding.Scopes.Range(func(_ string, scope ScopeBinding) bool {
		index.Set(checkScopeKey{
			kind:     scope.Kind,
			parentID: scope.ParentID,
			pos:      scope.Pos,
			end:      scope.End,
		}, scope.ID)
		return true
	})
	return index
}

func (c *checker) scopeID(key checkScopeKey) string {
	id, _ := c.scopeIndex.Get(key)
	return id
}
