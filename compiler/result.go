package compiler

import (
	"context"
	"github.com/arcgolabs/plano/diag"
	"go/token"
)

type Result struct {
	Document    *Document
	Binding     *Binding
	Checks      *CheckInfo
	HIR         *HIR
	FileSet     *token.FileSet
	Diagnostics diag.Diagnostics
}

func (c *Compiler) CompileSource(ctx context.Context, filename string, src []byte) (*Document, diag.Diagnostics) {
	result := c.CompileSourceDetailed(ctx, filename, src)
	return result.Document, result.Diagnostics
}

func (c *Compiler) CompileFile(ctx context.Context, filename string) (*Document, diag.Diagnostics) {
	result := c.CompileFileDetailed(ctx, filename)
	return result.Document, result.Diagnostics
}

func (c *Compiler) CompileSourceDetailed(ctx context.Context, filename string, src []byte) Result {
	input := c.prepareSource(filename, src)
	index := c.bindUnits(input.units)
	input.diagnostics.Append(index.diagnostics)
	checks, checkDiags := c.checkUnits(input.units, index)
	input.diagnostics.Append(checkDiags)
	doc, hir, more := c.compileUnits(input.fileSet, input.units, index, checks)
	input.diagnostics.Append(more)
	return Result{
		Document:    doc,
		Binding:     index.binding,
		Checks:      checks,
		HIR:         hir,
		FileSet:     input.fileSet,
		Diagnostics: input.diagnostics,
	}
}

func (c *Compiler) CompileFileDetailed(ctx context.Context, filename string) Result {
	input := c.prepareFile(filename)
	index := c.bindUnits(input.units)
	input.diagnostics.Append(index.diagnostics)
	checks, checkDiags := c.checkUnits(input.units, index)
	input.diagnostics.Append(checkDiags)
	doc, hir, more := c.compileUnits(input.fileSet, input.units, index, checks)
	input.diagnostics.Append(more)
	return Result{
		Document:    doc,
		Binding:     index.binding,
		Checks:      checks,
		HIR:         hir,
		FileSet:     input.fileSet,
		Diagnostics: input.diagnostics,
	}
}
