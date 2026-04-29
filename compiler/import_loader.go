package compiler

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	collectiongraph "github.com/arcgolabs/collectionx/graph"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	planofrontend "github.com/arcgolabs/plano/frontend/plano"
	"github.com/samber/oops"
)

type importTraversal struct {
	compiler *Compiler
	fset     *token.FileSet
	graph    *collectiongraph.Graph[string, struct{}]
	units    *mapping.Map[string, parsedUnit]
	seen     *set.Set[string]
	stack    []string
	diags    diag.Diagnostics
}

func (c *Compiler) loadImportedUnits(fset *token.FileSet, root parsedUnit) ([]parsedUnit, diag.Diagnostics) {
	traversal := importTraversal{
		compiler: c,
		fset:     fset,
		graph:    collectiongraph.NewDirectedGraph[string, struct{}](),
		units:    mapping.NewMap[string, parsedUnit](),
		seen:     set.NewSet[string](root.Name),
	}
	traversal.addUnit(root)
	traversal.visit(root)

	order, err := traversal.graph.TopologicalSort()
	if err != nil {
		traversal.diags.AddError(token.NoPos, token.NoPos, oops.Wrapf(err, "sort import graph").Error())
		return nil, traversal.diags
	}

	imported := make([]parsedUnit, 0, max(len(order)-1, 0))
	for _, name := range order {
		if name == root.Name {
			continue
		}
		unit, ok := traversal.units.Get(name)
		if ok {
			imported = append(imported, unit)
		}
	}
	return imported, traversal.diags
}

func (t *importTraversal) visit(unit parsedUnit) {
	t.stack = append(t.stack, unit.Name)
	defer func() {
		t.stack = t.stack[:len(t.stack)-1]
	}()

	for _, stmt := range unit.File.Statements {
		imp, ok := stmt.(*ast.ImportDecl)
		if !ok {
			continue
		}
		paths, err := resolveImportPaths(unit.Name, imp.Path.Value)
		if err != nil {
			t.diags.AddError(imp.Pos(), imp.End(), err.Error())
			continue
		}
		for _, next := range paths {
			t.visitImport(imp, unit.Name, next)
		}
	}
}

func (t *importTraversal) visitImport(imp *ast.ImportDecl, importer, next string) {
	if index := stackIndex(t.stack, next); index >= 0 {
		t.diags.AddError(imp.Pos(), imp.End(), formatImportCycle(t.stack[index:], next))
		return
	}

	child, ok := t.units.Get(next)
	if !ok {
		src, err := t.compiler.ReadFile(next)
		if err != nil {
			t.diags.AddError(imp.Pos(), imp.End(), oops.Wrapf(err, "read import file %q", next).Error())
			return
		}
		file, parseDiags := planofrontend.ParseFile(t.fset, next, src)
		t.diags.Append(parseDiags)
		child = parsedUnit{Name: next, File: file}
		t.addUnit(child)
	}

	if err := t.graph.AddEdge(next, importer); err != nil {
		t.diags.AddError(imp.Pos(), imp.End(), oops.Wrapf(err, "add import edge %q -> %q", next, importer).Error())
		return
	}
	if t.seen.Contains(next) {
		return
	}
	t.seen.Add(next)
	t.visit(child)
}

func (t *importTraversal) addUnit(unit parsedUnit) {
	t.graph.AddNode(unit.Name, struct{}{})
	t.units.Set(unit.Name, unit)
}

func stackIndex(items []string, target string) int {
	for index, item := range items {
		if item == target {
			return index
		}
	}
	return -1
}

func formatImportCycle(path []string, next string) string {
	cycle := append(append([]string(nil), path...), next)
	return "import cycle detected: " + strings.Join(cycle, " -> ")
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
