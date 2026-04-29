package compiler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/arcgolabs/plano/compiler"
)

func TestCompilerParseCacheReusesPreparedInput(t *testing.T) {
	root := filepath.Join(t.TempDir(), "build.plano")
	writeSource(t, root, `
const project_name: string = "demo"

workspace {
  name = project_name
}
`)

	c := compiler.New(compiler.Options{ParseCacheEntries: 4})
	first := c.BindFileDetailed(context.Background(), root)
	second := c.BindFileDetailed(context.Background(), root)
	if first.FileSet != second.FileSet {
		t.Fatalf("expected parse cache hit to reuse fileset")
	}
}

func TestCompilerParseCacheInvalidatesOnImportChange(t *testing.T) {
	dir := t.TempDir()
	defs := filepath.Join(dir, "defs.plano")
	root := filepath.Join(dir, "build.plano")
	writeSource(t, defs, `const project_name: string = "demo"`)
	writeSource(t, root, `
import "./defs.plano"

workspace {
  name = project_name
}
`)

	c := compiler.New(compiler.Options{ParseCacheEntries: 4})
	first := c.BindFileDetailed(context.Background(), root)
	writeSource(t, defs, `const project_name: string = "demo-2"`)
	second := c.BindFileDetailed(context.Background(), root)
	if first.FileSet == second.FileSet {
		t.Fatalf("expected parse cache miss after import change")
	}
}

func TestCompilerParseCacheEvictsLeastRecentlyUsed(t *testing.T) {
	dir := t.TempDir()
	firstFile := filepath.Join(dir, "first.plano")
	secondFile := filepath.Join(dir, "second.plano")
	writeSource(t, firstFile, `workspace { name = "first" }`)
	writeSource(t, secondFile, `workspace { name = "second" }`)

	c := compiler.New(compiler.Options{ParseCacheEntries: 1})
	first := c.BindFileDetailed(context.Background(), firstFile)
	_ = c.BindFileDetailed(context.Background(), secondFile)
	again := c.BindFileDetailed(context.Background(), firstFile)
	if first.FileSet == again.FileSet {
		t.Fatalf("expected first entry to be evicted")
	}
}

func writeSource(t *testing.T, path, src string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
}
