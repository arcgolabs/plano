package compiler

import (
	"context"
	"go/token"
	"path/filepath"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	planofrontend "github.com/arcgolabs/plano/frontend/plano"
	"github.com/samber/lo"
	"github.com/samber/oops"
)

func (c *Compiler) BindSource(ctx context.Context, filename string, src []byte) (*Binding, diag.Diagnostics) {
	result := c.BindSourceDetailed(ctx, filename, src)
	return result.Binding, result.Diagnostics
}

func (c *Compiler) BindFile(ctx context.Context, filename string) (*Binding, diag.Diagnostics) {
	result := c.BindFileDetailed(ctx, filename)
	return result.Binding, result.Diagnostics
}

func (c *Compiler) BindSourceDetailed(ctx context.Context, filename string, src []byte) BindResult {
	_ = ctx
	input := c.prepareSource(filename, src)
	index := c.bindUnits(input.units)
	input.diagnostics.Append(index.diagnostics)
	return BindResult{
		Binding:     index.binding,
		FileSet:     input.fileSet,
		Diagnostics: input.diagnostics,
	}
}

func (c *Compiler) BindFileDetailed(ctx context.Context, filename string) BindResult {
	_ = ctx
	input := c.prepareFile(filename)
	index := c.bindUnits(input.units)
	input.diagnostics.Append(index.diagnostics)
	return BindResult{
		Binding:     index.binding,
		FileSet:     input.fileSet,
		Diagnostics: input.diagnostics,
	}
}

func (c *Compiler) prepareSource(filename string, src []byte) preparedInput {
	fset := token.NewFileSet()
	root, diags := planofrontend.ParseFile(fset, filename, src)
	units := []parsedUnit{{Name: filepath.Clean(filename), File: root}}
	imported, importDiags := c.loadImportedUnits(fset, units[0])
	diags.Append(importDiags)
	return preparedInput{
		fileSet:     fset,
		units:       append(imported, units...),
		diagnostics: diags,
	}
}

func (c *Compiler) prepareFile(filename string) preparedInput {
	fset := token.NewFileSet()
	clean := filepath.Clean(filename)

	src, err := c.ReadFile(clean)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError(token.NoPos, token.NoPos, oops.Wrapf(err, "read source file %q", clean).Error())
		return preparedInput{
			fileSet:     fset,
			diagnostics: diags,
		}
	}

	root, diags := planofrontend.ParseFile(fset, clean, src)
	units := []parsedUnit{{Name: clean, File: root}}
	imported, importDiags := c.loadImportedUnits(fset, units[0])
	diags.Append(importDiags)
	return preparedInput{
		fileSet:     fset,
		units:       append(imported, units...),
		diagnostics: diags,
	}
}

func (c *Compiler) bindUnits(units []parsedUnit) boundIndex {
	b := binder{
		compiler: c,
		binding: &Binding{
			Files:     unitNames(units),
			Scopes:    mapping.NewOrderedMap[string, ScopeBinding](),
			Locals:    mapping.NewOrderedMap[string, LocalBinding](),
			Uses:      mapping.NewOrderedMap[string, NameUse](),
			Symbols:   mapping.NewOrderedMap[string, Symbol](),
			Consts:    mapping.NewOrderedMap[string, ConstBinding](),
			Functions: mapping.NewOrderedMap[string, FunctionBinding](),
		},
		symbols:    mapping.NewOrderedMap[string, Symbol](),
		constDecls: mapping.NewOrderedMap[string, *ast.ConstDecl](),
		funcDecls:  mapping.NewOrderedMap[string, *ast.FnDecl](),
	}

	for _, unit := range units {
		b.bindUnit(unit)
	}
	b.resolveUnits(units)

	return boundIndex{
		binding:     b.binding,
		symbols:     b.symbols,
		constDecls:  b.constDecls,
		funcDecls:   b.funcDecls,
		diagnostics: b.diags,
	}
}

func (b *binder) bindUnit(unit parsedUnit) {
	for _, stmt := range unit.File.Statements {
		switch node := stmt.(type) {
		case *ast.ConstDecl:
			b.bindConst(node)
		case *ast.FnDecl:
			b.bindFunction(node)
		case *ast.FormDecl:
			b.bindFormSymbols(node)
		}
	}
}

func (b *binder) bindConst(decl *ast.ConstDecl) {
	name := decl.Name.Name
	if b.hasDefinition(name) {
		b.diags.AddError(decl.Pos(), decl.End(), `duplicate definition "`+name+`"`)
		return
	}
	b.constDecls.Set(name, decl)
	b.binding.Consts.Set(name, ConstBinding{
		Name: name,
		Type: convertTypeExpr(decl.Type),
		Pos:  decl.Name.Pos(),
		End:  decl.Name.End(),
	})
}

func (b *binder) bindFunction(decl *ast.FnDecl) {
	name := decl.Name.Name
	if b.hasDefinition(name) {
		b.diags.AddError(decl.Pos(), decl.End(), `duplicate definition "`+name+`"`)
		return
	}
	b.funcDecls.Set(name, decl)
	b.binding.Functions.Set(name, FunctionBinding{
		Name:   name,
		Params: bindParams(decl.Params),
		Result: convertTypeExpr(decl.Result),
		Pos:    decl.Name.Pos(),
		End:    decl.Name.End(),
	})
}

func (b *binder) bindFormSymbols(form *ast.FormDecl) {
	spec, ok := b.compiler.forms.Get(form.Head.String())
	if ok && spec.Declares != "" && form.Label != nil && !form.Label.Quoted {
		name := form.Label.Value
		if b.hasDefinition(name) {
			b.diags.AddError(form.Pos(), form.End(), `duplicate definition "`+name+`"`)
		} else {
			symbol := Symbol{
				Name: name,
				Kind: spec.Declares,
				Pos:  form.Label.Pos(),
				End:  form.Label.End(),
			}
			b.symbols.Set(name, symbol)
			b.binding.Symbols.Set(name, symbol)
		}
	}
	if form.Body == nil {
		return
	}
	for _, item := range form.Body.Items {
		nested, ok := item.(*ast.FormDecl)
		if ok {
			b.bindFormSymbols(nested)
		}
	}
}

func (b *binder) hasDefinition(name string) bool {
	if _, exists := b.compiler.globals.Get(name); exists {
		return true
	}
	if _, exists := b.symbols.Get(name); exists {
		return true
	}
	if _, exists := b.constDecls.Get(name); exists {
		return true
	}
	if _, exists := b.funcDecls.Get(name); exists {
		return true
	}
	return false
}

func bindParams(params []*ast.Param) list.List[ParamBinding] {
	items := lo.Map(params, func(param *ast.Param, _ int) ParamBinding {
		return ParamBinding{
			Name: param.Name.Name,
			Type: convertTypeExpr(param.Type),
			Pos:  param.Pos(),
			End:  param.End(),
		}
	})
	return *list.NewList(items...)
}

func unitNames(units []parsedUnit) list.List[string] {
	items := lo.Map(units, func(unit parsedUnit, _ int) string {
		return unit.Name
	})
	return *list.NewList(items...)
}
