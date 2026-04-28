package lsp

import (
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/compiler"
)

type Position struct {
	Line      int `json:"line"      yaml:"line"`
	Character int `json:"character" yaml:"character"`
}

type Range struct {
	Start Position `json:"start" yaml:"start"`
	End   Position `json:"end"   yaml:"end"`
}

type Location struct {
	URI   string `json:"uri"   yaml:"uri"`
	Range Range  `json:"range" yaml:"range"`
}

type Diagnostic struct {
	Severity string `json:"severity" yaml:"severity"`
	Message  string `json:"message"  yaml:"message"`
	Range    Range  `json:"range"    yaml:"range"`
}

type Hover struct {
	Range    Range  `json:"range"    yaml:"range"`
	Contents string `json:"contents" yaml:"contents"`
}

type Document struct {
	URI     string `json:"uri"     yaml:"uri"`
	Path    string `json:"path"    yaml:"path"`
	Version int32  `json:"version" yaml:"version"`
	Text    []byte `json:"-"       yaml:"-"`
}

type Options struct {
	Compiler    *compiler.Compiler
	NewCompiler func() *compiler.Compiler
}

type Snapshot struct {
	URI         string          `json:"uri"         yaml:"uri"`
	Path        string          `json:"path"        yaml:"path"`
	Version     int32           `json:"version"     yaml:"version"`
	Result      compiler.Result `json:"result"      yaml:"result"`
	Diagnostics []Diagnostic    `json:"diagnostics" yaml:"diagnostics"`
	compiler    *compiler.Compiler
	documents   *mapping.Map[string, Document]
	sources     *mapping.Map[string, []byte]
}

func (o Options) baseCompiler() *compiler.Compiler {
	switch {
	case o.NewCompiler != nil:
		if built := o.NewCompiler(); built != nil {
			return built
		}
	case o.Compiler != nil:
		return o.Compiler
	}
	return compiler.New(compiler.Options{})
}
