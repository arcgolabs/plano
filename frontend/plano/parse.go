// Package plano exposes the public parser entrypoint for .plano source files.
package plano

import (
	"go/token"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/internal/parser"
)

func ParseFile(fset *token.FileSet, filename string, src []byte) (*ast.File, diag.Diagnostics) {
	if fset == nil {
		fset = token.NewFileSet()
	}
	file := fset.AddFile(filename, -1, len(src))
	file.SetLinesForContent(src)
	return parser.Parse(file, src)
}
