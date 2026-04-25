package compiler

import (
	"context"
	"go/token"
	"os"
	"path/filepath"

	"github.com/arcgolabs/plano/diag"
	planofrontend "github.com/arcgolabs/plano/frontend/plano"
	"github.com/samber/oops"
)

type Result struct {
	Document    *Document
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
	_ = ctx
	fset := token.NewFileSet()
	root, diags := planofrontend.ParseFile(fset, filename, src)
	units := []parsedUnit{{Name: filepath.Clean(filename), File: root}}
	imported, importDiags := c.loadImports(fset, units[0], map[string]bool{filepath.Clean(filename): true}, map[string]bool{})
	diags.Append(importDiags)
	units = append(imported, units...)
	doc, more := c.compileUnits(fset, units)
	diags.Append(more)
	return Result{
		Document:    doc,
		FileSet:     fset,
		Diagnostics: diags,
	}
}

func (c *Compiler) CompileFileDetailed(ctx context.Context, filename string) Result {
	_ = ctx
	fset := token.NewFileSet()
	clean := filepath.Clean(filename)
	src, err := os.ReadFile(clean)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError(token.NoPos, token.NoPos, oops.Wrapf(err, "read source file %q", clean).Error())
		return Result{
			Document:    nil,
			FileSet:     fset,
			Diagnostics: diags,
		}
	}
	root, diags := planofrontend.ParseFile(fset, clean, src)
	units := []parsedUnit{{Name: clean, File: root}}
	imported, importDiags := c.loadImports(fset, units[0], map[string]bool{clean: true}, map[string]bool{})
	diags.Append(importDiags)
	units = append(imported, units...)
	doc, more := c.compileUnits(fset, units)
	diags.Append(more)
	return Result{
		Document:    doc,
		FileSet:     fset,
		Diagnostics: diags,
	}
}
