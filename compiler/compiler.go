package compiler

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	goruntime "runtime"

	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

type Options struct {
	LookupEnv func(string) (string, bool)
}

type Compiler struct {
	forms     *mapping.OrderedMap[string, schema.FormSpec]
	funcs     *mapping.OrderedMap[string, schema.FunctionSpec]
	actions   *mapping.OrderedMap[string, ActionSpec]
	globals   *mapping.OrderedMap[string, any]
	lookupEnv func(string) (string, bool)
}

type Document struct {
	Forms   []Form                              `json:"forms"   yaml:"forms"`
	Symbols *mapping.OrderedMap[string, Symbol] `json:"symbols" yaml:"symbols"`
	Consts  *mapping.OrderedMap[string, any]    `json:"consts"  yaml:"consts"`
}

type FormLabel struct {
	Kind  schema.LabelKind
	Value string
}

type Symbol struct {
	Name string
	Kind string
	Pos  token.Pos
}

type Call struct {
	Name string
	Args []any
	Pos  token.Pos
	End  token.Pos
}

type Form struct {
	Kind   string
	Label  *FormLabel
	Symbol *Symbol
	Fields *mapping.OrderedMap[string, any]
	Forms  []Form
	Calls  []Call
	Pos    token.Pos
	End    token.Pos
}

type parsedUnit struct {
	Name string
	File *ast.File
}

type compileState struct {
	compiler    *Compiler
	binding     *Binding
	checks      *CheckInfo
	hir         *HIR
	fset        *token.FileSet
	diags       diag.Diagnostics
	symbols     *mapping.OrderedMap[string, Symbol]
	constDecls  *mapping.OrderedMap[string, *ast.ConstDecl]
	constValues *mapping.OrderedMap[string, any]
	funcDecls   *mapping.OrderedMap[string, *ast.FnDecl]
	resolving   *mapping.OrderedMap[string, bool]
	scopeIndex  map[scopeSpanKey]string
	fieldIndex  map[spanKey]FieldCheck
	callIndex   map[spanKey]CallCheck
}

func New(opts Options) *Compiler {
	lookupEnv := opts.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	c := &Compiler{
		forms:     mapping.NewOrderedMap[string, schema.FormSpec](),
		funcs:     mapping.NewOrderedMap[string, schema.FunctionSpec](),
		actions:   mapping.NewOrderedMap[string, ActionSpec](),
		globals:   mapping.NewOrderedMap[string, any](),
		lookupEnv: lookupEnv,
	}
	c.RegisterConst("os", goruntime.GOOS)
	c.RegisterConst("arch", goruntime.GOARCH)
	c.registerBuiltins()
	return c
}

func (c *Compiler) RegisterForm(spec schema.FormSpec) error {
	if spec.Name == "" {
		return errors.New("form name cannot be empty")
	}
	if spec.Fields == nil {
		spec.Fields = make(map[string]schema.FieldSpec)
	}
	if spec.NestedForms == nil {
		spec.NestedForms = make(map[string]struct{})
	}
	c.forms.Set(spec.Name, spec)
	return nil
}

func (c *Compiler) RegisterFunc(spec schema.FunctionSpec) error {
	if spec.Name == "" {
		return errors.New("function name cannot be empty")
	}
	if spec.Eval == nil {
		return fmt.Errorf("function %q has nil evaluator", spec.Name)
	}
	c.funcs.Set(spec.Name, spec)
	return nil
}

func (c *Compiler) RegisterConst(name string, value any) {
	c.globals.Set(name, value)
}

func (c *Compiler) compileUnits(fset *token.FileSet, units []parsedUnit, index boundIndex, checks *CheckInfo) (*Document, *HIR, diag.Diagnostics) {
	state := c.newCompileState(fset, index, checks)
	state.resolveAllConsts()
	state.populateHIRConsts()
	doc := state.newDocument()
	state.compileTopLevelForms(units, doc)
	return doc, state.hir, state.diags
}
