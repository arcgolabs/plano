//nolint:cyclop,gocognit,gocyclo,funlen,revive // Core compiler state and dispatch stay centralized while the language surface is still moving.
package compiler

import (
	"errors"
	"fmt"
	"go/token"
	"maps"
	"os"
	goruntime "runtime"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/diag"
	planofrontend "github.com/arcgolabs/plano/frontend/plano"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/oops"
)

type Options struct {
	LookupEnv func(string) (string, bool)
}

type Compiler struct {
	forms     map[string]schema.FormSpec
	funcs     map[string]schema.FunctionSpec
	actions   map[string]ActionSpec
	globals   map[string]any
	lookupEnv func(string) (string, bool)
}

type Document struct {
	Forms   []Form
	Symbols map[string]Symbol
	Consts  map[string]any
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
	Fields map[string]any
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
	fset        *token.FileSet
	diags       diag.Diagnostics
	symbols     map[string]Symbol
	constDecls  map[string]*ast.ConstDecl
	constValues map[string]any
	funcDecls   map[string]*ast.FnDecl
	resolving   map[string]bool
}

func New(opts Options) *Compiler {
	lookupEnv := opts.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	c := &Compiler{
		forms:     make(map[string]schema.FormSpec),
		funcs:     make(map[string]schema.FunctionSpec),
		actions:   make(map[string]ActionSpec),
		globals:   make(map[string]any),
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
	c.forms[spec.Name] = spec
	return nil
}

func (c *Compiler) RegisterFunc(spec schema.FunctionSpec) error {
	if spec.Name == "" {
		return errors.New("function name cannot be empty")
	}
	if spec.Eval == nil {
		return fmt.Errorf("function %q has nil evaluator", spec.Name)
	}
	c.funcs[spec.Name] = spec
	return nil
}

func (c *Compiler) RegisterConst(name string, value any) {
	c.globals[name] = value
}

func (c *Compiler) loadImports(fset *token.FileSet, unit parsedUnit, seen, stack map[string]bool) ([]parsedUnit, diag.Diagnostics) {
	var diags diag.Diagnostics
	stack[unit.Name] = true
	defer delete(stack, unit.Name)

	var out []parsedUnit
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
			if stack[next] {
				diags.AddError(imp.Pos(), imp.End(), "import cycle detected involving "+next)
				continue
			}
			if seen[next] {
				continue
			}
			seen[next] = true

			//nolint:gosec // Import resolution intentionally reads source files chosen by the DSL import graph.
			src, err := os.ReadFile(next)
			if err != nil {
				diags.AddError(imp.Pos(), imp.End(), oops.Wrapf(err, "read import file %q", next).Error())
				continue
			}
			file, parseDiags := planofrontend.ParseFile(fset, next, src)
			diags.Append(parseDiags)
			child := parsedUnit{Name: next, File: file}
			nested, nestedDiags := c.loadImports(fset, child, seen, stack)
			diags.Append(nestedDiags)
			out = append(out, nested...)
			out = append(out, child)
		}
	}
	return out, diags
}

func (c *Compiler) compileUnits(fset *token.FileSet, units []parsedUnit) (*Document, diag.Diagnostics) {
	state := &compileState{
		compiler:    c,
		fset:        fset,
		symbols:     make(map[string]Symbol),
		constDecls:  make(map[string]*ast.ConstDecl),
		constValues: make(map[string]any),
		funcDecls:   make(map[string]*ast.FnDecl),
		resolving:   make(map[string]bool),
	}

	for _, unit := range units {
		for _, stmt := range unit.File.Statements {
			switch node := stmt.(type) {
			case *ast.ConstDecl:
				state.collectConst(node)
			case *ast.FnDecl:
				state.collectFunction(node)
			case *ast.FormDecl:
				state.collectSymbols(node)
			}
		}
	}

	for name := range state.constDecls {
		state.resolveConst(name)
	}

	doc := &Document{
		Symbols: make(map[string]Symbol, len(state.symbols)),
		Consts:  make(map[string]any, len(state.constValues)),
	}
	maps.Copy(doc.Symbols, state.symbols)
	maps.Copy(doc.Consts, state.constValues)

	for _, unit := range units {
		for _, stmt := range unit.File.Statements {
			form, ok := stmt.(*ast.FormDecl)
			if !ok {
				if _, unsupported := stmt.(*ast.ImportDecl); unsupported {
					continue
				}
				if _, isConst := stmt.(*ast.ConstDecl); isConst {
					continue
				}
				if _, isFn := stmt.(*ast.FnDecl); isFn {
					continue
				}
				state.diags.AddError(stmt.Pos(), stmt.End(), "top-level statement is not supported by the compiler yet")
				continue
			}
			compiled := state.compileForm(form, nil)
			if compiled != nil {
				doc.Forms = append(doc.Forms, *compiled)
			}
		}
	}

	return doc, state.diags
}

func (s *compileState) collectConst(decl *ast.ConstDecl) {
	name := decl.Name.Name
	if s.hasDefinition(name) {
		s.diags.AddError(decl.Pos(), decl.End(), fmt.Sprintf("duplicate definition %q", name))
		return
	}
	s.constDecls[name] = decl
}

func (s *compileState) collectFunction(decl *ast.FnDecl) {
	name := decl.Name.Name
	if s.hasDefinition(name) {
		s.diags.AddError(decl.Pos(), decl.End(), fmt.Sprintf("duplicate definition %q", name))
		return
	}
	s.funcDecls[name] = decl
}

func (s *compileState) collectSymbols(form *ast.FormDecl) {
	spec, ok := s.compiler.forms[form.Head.String()]
	if ok && spec.Declares != "" && form.Label != nil && !form.Label.Quoted {
		if s.hasDefinition(form.Label.Value) {
			s.diags.AddError(form.Pos(), form.End(), fmt.Sprintf("duplicate definition %q", form.Label.Value))
		} else {
			s.symbols[form.Label.Value] = Symbol{
				Name: form.Label.Value,
				Kind: spec.Declares,
				Pos:  form.Label.Pos(),
			}
		}
	}
	if form.Body == nil {
		return
	}
	for _, item := range form.Body.Items {
		nested, ok := item.(*ast.FormDecl)
		if ok {
			s.collectSymbols(nested)
		}
	}
}

func (s *compileState) hasDefinition(name string) bool {
	if _, exists := s.compiler.globals[name]; exists {
		return true
	}
	if _, exists := s.symbols[name]; exists {
		return true
	}
	if _, exists := s.constDecls[name]; exists {
		return true
	}
	if _, exists := s.funcDecls[name]; exists {
		return true
	}
	return false
}

func (s *compileState) resolveConst(name string) (any, bool) {
	if value, ok := s.constValues[name]; ok {
		return value, true
	}
	decl, ok := s.constDecls[name]
	if !ok {
		return nil, false
	}
	if s.resolving[name] {
		s.diags.AddError(decl.Pos(), decl.End(), fmt.Sprintf("constant cycle detected at %q", name))
		return nil, false
	}
	s.resolving[name] = true
	value, err := s.evalExpr(decl.Value, nil)
	delete(s.resolving, name)
	if err != nil {
		s.diags.AddError(decl.Pos(), decl.End(), err.Error())
		return nil, false
	}
	if decl.Type != nil {
		typ := s.convertType(decl.Type)
		if typ != nil {
			if err := schema.CheckAssignable(typ, value); err != nil {
				s.diags.AddError(decl.Pos(), decl.End(), fmt.Sprintf("const %q: %v", name, err))
				return nil, false
			}
		}
	}
	s.constValues[name] = value
	return value, true
}

func (s *compileState) compileForm(node *ast.FormDecl, locals *env) *Form {
	spec, ok := s.compiler.forms[node.Head.String()]
	if !ok {
		s.diags.AddError(node.Pos(), node.End(), fmt.Sprintf("unknown form %q", node.Head.String()))
		return nil
	}

	out := &Form{
		Kind:   spec.Name,
		Fields: make(map[string]any),
		Pos:    node.Pos(),
		End:    node.End(),
	}
	if node.Label != nil {
		out.Label = &FormLabel{
			Kind:  spec.LabelKind,
			Value: node.Label.Value,
		}
	}
	switch spec.LabelKind {
	case schema.LabelNone:
		if node.Label != nil {
			s.diags.AddError(node.Label.Pos(), node.Label.End(), spec.Name+" does not accept label")
		}
	case schema.LabelSymbol:
		if node.Label == nil {
			s.diags.AddError(node.Pos(), node.End(), spec.Name+" requires symbol label")
		} else if node.Label.Quoted {
			s.diags.AddError(node.Label.Pos(), node.Label.End(), spec.Name+" requires identifier label")
		} else if symbol, ok := s.symbols[node.Label.Value]; ok {
			sym := symbol
			out.Symbol = &sym
		}
	case schema.LabelString:
		if node.Label == nil {
			s.diags.AddError(node.Pos(), node.End(), spec.Name+" requires string label")
		} else if !node.Label.Quoted {
			s.diags.AddError(node.Label.Pos(), node.Label.End(), spec.Name+" requires string label")
		}
	}

	fieldSeen := make(map[string]bool)
	formEnv := newEnv(locals)
	s.execFormItems(&formExecState{
		spec:      spec,
		form:      out,
		fieldSeen: fieldSeen,
	}, node.Body.Items, formEnv)

	for name, field := range spec.Fields {
		if fieldSeen[name] {
			continue
		}
		if field.HasDefault {
			out.Fields[name] = field.Default
			continue
		}
		if field.Required {
			s.diags.AddError(node.Pos(), node.End(), fmt.Sprintf("%s requires field %q", spec.Name, name))
		}
	}

	return out
}

//nolint:nilnil // nil is the runtime representation of the DSL null literal.
func (s *compileState) evalExpr(expr ast.Expr, locals *env) (any, error) {
	switch node := expr.(type) {
	case *ast.StringLiteral:
		return node.Value, nil
	case *ast.IntLiteral:
		return node.Value, nil
	case *ast.FloatLiteral:
		return node.Value, nil
	case *ast.BoolLiteral:
		return node.Value, nil
	case *ast.NullLiteral:
		return nil, nil
	case *ast.DurationLiteral:
		value, err := schema.ParseDuration(node.Raw)
		if err != nil {
			return nil, fmt.Errorf("parse duration %q: %w", node.Raw, err)
		}
		return value, nil
	case *ast.SizeLiteral:
		value, err := schema.ParseSize(node.Raw)
		if err != nil {
			return nil, fmt.Errorf("parse size %q: %w", node.Raw, err)
		}
		return value, nil
	case *ast.IdentExpr:
		return s.resolveName(node.Name.Name, locals)
	case *ast.ArrayExpr:
		items := make([]any, 0, len(node.Elements))
		for _, elem := range node.Elements {
			value, err := s.evalExpr(elem, locals)
			if err != nil {
				return nil, err
			}
			items = append(items, value)
		}
		return items, nil
	case *ast.ObjectExpr:
		items := make(map[string]any, len(node.Entries))
		for _, entry := range node.Entries {
			value, err := s.evalExpr(entry.Value, locals)
			if err != nil {
				return nil, err
			}
			items[entry.Key.Name] = value
		}
		return items, nil
	case *ast.ParenExpr:
		return s.evalExpr(node.X, locals)
	case *ast.UnaryExpr:
		value, err := s.evalExpr(node.X, locals)
		if err != nil {
			return nil, err
		}
		return evalUnary(node.Op, value)
	case *ast.BinaryExpr:
		left, err := s.evalExpr(node.X, locals)
		if err != nil {
			return nil, err
		}
		right, err := s.evalExpr(node.Y, locals)
		if err != nil {
			return nil, err
		}
		return evalBinary(node.Op, left, right)
	case *ast.SelectorExpr:
		base, err := s.evalExpr(node.X, locals)
		if err != nil {
			return nil, err
		}
		object, ok := base.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("selector requires object, got %T", base)
		}
		value, ok := object[node.Sel.Name]
		if !ok {
			return nil, fmt.Errorf("unknown field %q", node.Sel.Name)
		}
		return value, nil
	case *ast.IndexExpr:
		base, err := s.evalExpr(node.X, locals)
		if err != nil {
			return nil, err
		}
		index, err := s.evalExpr(node.Index, locals)
		if err != nil {
			return nil, err
		}
		return evalIndex(base, index)
	case *ast.CallExpr:
		name, ok := callName(node.Fun)
		if !ok {
			return nil, errors.New("unsupported call target")
		}
		args := make([]any, 0, len(node.Args))
		for _, arg := range node.Args {
			value, err := s.evalExpr(arg, locals)
			if err != nil {
				return nil, err
			}
			args = append(args, value)
		}
		if spec, ok := s.compiler.funcs[name]; ok {
			if err := validateArity("function", name, spec.MinArgs, spec.MaxArgs, len(args)); err != nil {
				return nil, err
			}
			value, err := spec.Eval(args)
			if err != nil {
				return nil, fmt.Errorf("evaluate function %q: %w", name, err)
			}
			return value, nil
		}
		if decl, ok := s.funcDecls[name]; ok {
			return s.callUserFunction(name, decl, args)
		}
		return nil, fmt.Errorf("unknown function %q", name)
	default:
		return nil, fmt.Errorf("unsupported expression %T", expr)
	}
}

func (s *compileState) resolveName(name string, locals *env) (any, error) {
	if locals != nil {
		if value, ok := locals.Get(name); ok {
			return value, nil
		}
	}
	if value, ok := s.compiler.globals[name]; ok {
		return value, nil
	}
	if value, ok := s.constValues[name]; ok {
		return value, nil
	}
	if _, ok := s.constDecls[name]; ok {
		value, ok := s.resolveConst(name)
		if ok {
			return value, nil
		}
		return nil, fmt.Errorf("failed to resolve constant %q", name)
	}
	if symbol, ok := s.symbols[name]; ok {
		return schema.Ref{Kind: symbol.Kind, Name: symbol.Name}, nil
	}
	return nil, fmt.Errorf("undefined symbol %q", name)
}

func (s *compileState) convertType(node ast.TypeExpr) schema.Type {
	switch current := node.(type) {
	case *ast.SimpleType:
		switch current.Name.Name {
		case "string":
			return schema.TypeString
		case "int":
			return schema.TypeInt
		case "float":
			return schema.TypeFloat
		case "bool":
			return schema.TypeBool
		case "duration":
			return schema.TypeDuration
		case "size":
			return schema.TypeSize
		case "path":
			return schema.TypePath
		case "any":
			return schema.TypeAny
		default:
			return schema.NamedType{Name: current.Name.Name}
		}
	case *ast.QualifiedType:
		return schema.NamedType{Name: current.Name.String()}
	case *ast.ListType:
		return schema.ListType{Elem: s.convertType(current.Elem)}
	case *ast.MapType:
		return schema.MapType{Elem: s.convertType(current.Elem)}
	case *ast.RefType:
		return schema.RefType{Kind: current.Target.String()}
	default:
		return nil
	}
}

func allowsField(mode schema.BodyMode) bool {
	return mode == schema.BodyFieldOnly || mode == schema.BodyMixed || mode == schema.BodyScript
}

func allowsForm(mode schema.BodyMode) bool {
	return mode == schema.BodyFormOnly || mode == schema.BodyMixed || mode == schema.BodyScript
}

func allowsCall(mode schema.BodyMode) bool {
	return mode == schema.BodyCallOnly || mode == schema.BodyScript
}
