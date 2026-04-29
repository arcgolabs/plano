package compiler

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	goruntime "runtime"

	"github.com/arcgolabs/collectionx/list"
	"github.com/arcgolabs/collectionx/mapping"
	"github.com/arcgolabs/collectionx/set"
	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	"github.com/arcgolabs/plano/schema"
)

type Options struct {
	LookupEnv func(string) (string, bool)
	ReadFile  func(string) ([]byte, error)
}

type Compiler struct {
	forms     *mapping.OrderedMap[string, schema.FormSpec]
	funcs     *mapping.OrderedMap[string, schema.FunctionSpec]
	actions   *mapping.OrderedMap[string, ActionSpec]
	globals   *mapping.OrderedMap[string, any]
	lookupEnv func(string) (string, bool)
	readFile  func(string) ([]byte, error)
}

type Document struct {
	Forms   list.List[Form]                     `json:"forms"   yaml:"forms"`
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
	End  token.Pos
}

type Call struct {
	Name string
	Args list.List[any]
	Pos  token.Pos
	End  token.Pos
}

type Form struct {
	Kind   string
	Label  *FormLabel
	Symbol *Symbol
	Fields *mapping.OrderedMap[string, any]
	Forms  list.List[Form]
	Calls  list.List[Call]
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
	scopeIndex  *mapping.Map[scopeSpanKey, string]
	fieldIndex  *mapping.Map[spanKey, FieldCheck]
	callIndex   *mapping.Map[spanKey, CallCheck]
}

func New(opts Options) *Compiler {
	lookupEnv := opts.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	readFile := opts.ReadFile
	if readFile == nil {
		readFile = readSourceFile
	}

	c := &Compiler{
		forms:     mapping.NewOrderedMap[string, schema.FormSpec](),
		funcs:     mapping.NewOrderedMap[string, schema.FunctionSpec](),
		actions:   mapping.NewOrderedMap[string, ActionSpec](),
		globals:   mapping.NewOrderedMap[string, any](),
		lookupEnv: lookupEnv,
		readFile:  readFile,
	}
	c.RegisterConst("os", goruntime.GOOS)
	c.RegisterConst("arch", goruntime.GOARCH)
	c.registerBuiltins()
	return c
}

func (c *Compiler) Clone() *Compiler {
	if c == nil {
		return New(Options{})
	}
	return &Compiler{
		forms:     c.forms.Clone(),
		funcs:     c.funcs.Clone(),
		actions:   c.actions.Clone(),
		globals:   c.globals.Clone(),
		lookupEnv: c.lookupEnv,
		readFile:  c.readFile,
	}
}

func (c *Compiler) SetReadFile(fn func(string) ([]byte, error)) {
	if c == nil {
		return
	}
	if fn == nil {
		c.readFile = readSourceFile
		return
	}
	c.readFile = fn
}

func (c *Compiler) ReadFile(path string) ([]byte, error) {
	if c == nil || c.readFile == nil {
		return readSourceFile(path)
	}
	return c.readFile(path)
}

func (c *Compiler) FormSpec(name string) (schema.FormSpec, bool) {
	if c == nil || c.forms == nil {
		return schema.FormSpec{}, false
	}
	return c.forms.Get(name)
}

func (c *Compiler) FormSpecs() *mapping.OrderedMap[string, schema.FormSpec] {
	if c == nil || c.forms == nil {
		return mapping.NewOrderedMap[string, schema.FormSpec]()
	}
	return c.forms.Clone()
}

func (c *Compiler) FunctionSpec(name string) (schema.FunctionSpec, bool) {
	if c == nil || c.funcs == nil {
		return schema.FunctionSpec{}, false
	}
	return c.funcs.Get(name)
}

func (c *Compiler) FunctionSpecs() *mapping.OrderedMap[string, schema.FunctionSpec] {
	if c == nil || c.funcs == nil {
		return mapping.NewOrderedMap[string, schema.FunctionSpec]()
	}
	return c.funcs.Clone()
}

func (c *Compiler) ActionSpec(name string) (ActionSpec, bool) {
	if c == nil || c.actions == nil {
		return ActionSpec{}, false
	}
	return c.actions.Get(name)
}

func (c *Compiler) ActionSpecs() *mapping.OrderedMap[string, ActionSpec] {
	if c == nil || c.actions == nil {
		return mapping.NewOrderedMap[string, ActionSpec]()
	}
	return c.actions.Clone()
}

func (c *Compiler) Globals() *mapping.OrderedMap[string, any] {
	if c == nil || c.globals == nil {
		return mapping.NewOrderedMap[string, any]()
	}
	return c.globals.Clone()
}

func (c *Compiler) RegisterForm(spec schema.FormSpec) error {
	if spec.Name == "" {
		return errors.New("form name cannot be empty")
	}
	if spec.Fields == nil {
		spec.Fields = mapping.NewOrderedMap[string, schema.FieldSpec]()
	} else {
		fields := mapping.NewOrderedMapWithCapacity[string, schema.FieldSpec](spec.Fields.Len())
		spec.Fields.Range(func(name string, field schema.FieldSpec) bool {
			if field.Name == "" {
				field.Name = name
			}
			fields.Set(name, field)
			return true
		})
		spec.Fields = fields
	}
	if spec.NestedForms == nil {
		spec.NestedForms = set.NewSet[string]()
	} else {
		spec.NestedForms = spec.NestedForms.Clone()
	}
	c.forms.Set(spec.Name, spec)
	return nil
}

func (c *Compiler) RegisterForms(specs list.List[schema.FormSpec]) error {
	for idx := range specs.Len() {
		spec, _ := specs.Get(idx)
		if err := c.RegisterForm(spec); err != nil {
			return fmt.Errorf("register form %q: %w", spec.Name, err)
		}
	}
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
