package compiler

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	planofrontend "github.com/arcgolabs/plano/frontend/plano"
	"github.com/samber/oops"
)

func (c *Compiler) loadImports(fset *token.FileSet, unit parsedUnit, seen, stack map[string]bool) ([]parsedUnit, diag.Diagnostics) {
	var diags diag.Diagnostics
	stack[unit.Name] = true
	defer delete(stack, unit.Name)

	out := make([]parsedUnit, 0)
	for _, stmt := range unit.File.Statements {
		imp, ok := stmt.(*ast.ImportDecl)
		if !ok {
			continue
		}
		paths, err := resolveImportPaths(unit.Name, imp.Path.Value)
		if err != nil {
			diags.AddError(imp.Pos(), imp.End(), err.Error())
			continue
		}
		for _, next := range paths {
			imported, importDiags := c.loadImportUnit(fset, imp, next, seen, stack)
			diags.Append(importDiags)
			out = append(out, imported...)
		}
	}
	return out, diags
}

func (c *Compiler) loadImportUnit(
	fset *token.FileSet,
	imp *ast.ImportDecl,
	next string,
	seen map[string]bool,
	stack map[string]bool,
) ([]parsedUnit, diag.Diagnostics) {
	var diags diag.Diagnostics
	if stack[next] {
		diags.AddError(imp.Pos(), imp.End(), "import cycle detected involving "+next)
		return nil, diags
	}
	if seen[next] {
		return nil, diags
	}
	seen[next] = true

	src, err := c.ReadFile(next)
	if err != nil {
		diags.AddError(imp.Pos(), imp.End(), oops.Wrapf(err, "read import file %q", next).Error())
		return nil, diags
	}
	file, parseDiags := planofrontend.ParseFile(fset, next, src)
	diags.Append(parseDiags)

	child := parsedUnit{Name: next, File: file}
	nested, nestedDiags := c.loadImports(fset, child, seen, stack)
	diags.Append(nestedDiags)
	return append(nested, child), diags
}

func readSourceFile(path string) ([]byte, error) {
	root, name, err := openSourceRoot(path)
	if err != nil {
		return nil, err
	}
	data, err := root.ReadFile(name)
	closeErr := root.Close()
	if err != nil {
		if closeErr != nil {
			return nil, oops.Wrapf(errors.Join(err, closeErr), "read and close import file %q", path)
		}
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close root for %q: %w", path, closeErr)
	}
	return data, nil
}

func openSourceRoot(path string) (*os.Root, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("resolve %q: %w", path, err)
	}
	root, err := os.OpenRoot(filepath.Dir(abs))
	if err != nil {
		return nil, "", fmt.Errorf("open root for %q: %w", path, err)
	}
	return root, filepath.Base(abs), nil
}
