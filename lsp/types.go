package lsp

import (
	"go/token"

	"github.com/arcgolabs/collectionx/interval"
	"github.com/arcgolabs/collectionx/list"
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
	Severity string                                  `json:"severity"       yaml:"severity"`
	Code     string                                  `json:"code,omitempty" yaml:"code,omitempty"`
	Message  string                                  `json:"message"        yaml:"message"`
	Range    Range                                   `json:"range"          yaml:"range"`
	Related  list.List[DiagnosticRelatedInformation] `json:"related"        yaml:"related"`
}

type DiagnosticRelatedInformation struct {
	Message  string   `json:"message"  yaml:"message"`
	Location Location `json:"location" yaml:"location"`
}

type Hover struct {
	Range    Range  `json:"range"    yaml:"range"`
	Contents string `json:"contents" yaml:"contents"`
}

type CompletionKind string

const (
	CompletionKeyword  CompletionKind = "keyword"
	CompletionForm     CompletionKind = "form"
	CompletionField    CompletionKind = "field"
	CompletionFunction CompletionKind = "function"
	CompletionAction   CompletionKind = "action"
	CompletionLocal    CompletionKind = "local"
	CompletionConst    CompletionKind = "const"
	CompletionSymbol   CompletionKind = "symbol"
	CompletionGlobal   CompletionKind = "global"
)

type CompletionItem struct {
	Label         string         `json:"label"         yaml:"label"`
	Kind          CompletionKind `json:"kind"          yaml:"kind"`
	Detail        string         `json:"detail"        yaml:"detail"`
	Documentation string         `json:"documentation" yaml:"documentation"`
}

type CompletionList struct {
	Range Range                     `json:"range" yaml:"range"`
	Items list.List[CompletionItem] `json:"items" yaml:"items"`
}

type TextEdit struct {
	Range   Range  `json:"range"   yaml:"range"`
	NewText string `json:"newText" yaml:"newText"`
}

type WorkspaceEdit struct {
	Changes *mapping.OrderedMap[string, list.List[TextEdit]] `json:"changes" yaml:"changes"`
}

type SymbolKind string

const (
	SymbolForm     SymbolKind = "form"
	SymbolFunction SymbolKind = "function"
	SymbolConst    SymbolKind = "const"
	SymbolField    SymbolKind = "field"
)

type DocumentSymbol struct {
	Name           string                    `json:"name"           yaml:"name"`
	Detail         string                    `json:"detail"         yaml:"detail"`
	Kind           SymbolKind                `json:"kind"           yaml:"kind"`
	Range          Range                     `json:"range"          yaml:"range"`
	SelectionRange Range                     `json:"selectionRange" yaml:"selectionRange"`
	Children       list.List[DocumentSymbol] `json:"children"       yaml:"children"`
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
	URI         string                `json:"uri"         yaml:"uri"`
	Path        string                `json:"path"        yaml:"path"`
	Version     int32                 `json:"version"     yaml:"version"`
	Result      compiler.Result       `json:"result"      yaml:"result"`
	Diagnostics list.List[Diagnostic] `json:"diagnostics" yaml:"diagnostics"`
	compiler    *compiler.Compiler
	documents   *mapping.Map[string, Document]
	files       *mapping.Map[string, *token.File]
	fileSpans   *interval.RangeMap[int, fileSpan]
	queries     *snapshotQueryCache
	sources     *mapping.Map[string, []byte]
}

type fileSpan struct {
	path string
	file *token.File
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
