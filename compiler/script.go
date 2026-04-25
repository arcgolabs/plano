//nolint:cyclop,gocognit,gocyclo,funlen,revive // Script execution keeps interpreter dispatch together for readability.
package compiler

import (
	"errors"
	"fmt"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
	"github.com/samber/mo"
)

type env struct {
	parent *env
	values map[string]any
}

func newEnv(parent *env) *env {
	return &env{
		parent: parent,
		values: make(map[string]any),
	}
}

func (e *env) Bind(name string, value any) {
	e.values[name] = value
}

func (e *env) Get(name string) (any, bool) {
	for current := e; current != nil; current = current.parent {
		if value, ok := current.values[name]; ok {
			return value, true
		}
	}
	return nil, false
}

type formExecState struct {
	spec      schema.FormSpec
	form      *Form
	fieldSeen map[string]bool
}

func (s *compileState) execFormItems(state *formExecState, items []ast.FormItem, locals *env) {
	for _, item := range items {
		switch current := item.(type) {
		case *ast.Assignment:
			s.execFieldAssignment(state, current, locals)
		case *ast.FormDecl:
			s.execNestedForm(state, current, locals)
		case *ast.CallStmt:
			s.execCall(state, current, locals)
		case *ast.ConstDecl:
			s.execLocalBinding(state, current.Name.Name, current.Type, current.Value, locals)
		case *ast.LetDecl:
			s.execLocalBinding(state, current.Name.Name, current.Type, current.Value, locals)
		case *ast.IfStmt:
			s.execIf(state, current, locals)
		case *ast.ForStmt:
			s.execFor(state, current, locals)
		case *ast.ReturnStmt:
			s.diags.AddError(current.Pos(), current.End(), "return is not allowed in form bodies")
		case *ast.FnDecl:
			s.diags.AddError(current.Pos(), current.End(), "nested function declarations are not implemented")
		case *ast.ImportDecl:
			s.diags.AddError(current.Pos(), current.End(), "import is not allowed in form bodies")
		default:
			s.diags.AddError(item.Pos(), item.End(), fmt.Sprintf("unsupported form item %T", item))
		}
	}
}

func (s *compileState) execFieldAssignment(state *formExecState, current *ast.Assignment, locals *env) {
	if !allowsField(state.spec.BodyMode) {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow fields in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	fieldSpec, ok := state.spec.Fields[current.Name.Name]
	if !ok {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("field %q is not allowed in %s", current.Name.Name, state.spec.Name))
		return
	}
	value, err := s.evalExpr(current.Value, locals)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return
	}
	if err := schema.CheckAssignable(fieldSpec.Type, value); err != nil {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("field %q: %v", fieldSpec.Name, err))
		return
	}
	state.form.Fields[current.Name.Name] = value
	state.fieldSeen[current.Name.Name] = true
}

func (s *compileState) execNestedForm(state *formExecState, current *ast.FormDecl, locals *env) {
	if !allowsForm(state.spec.BodyMode) {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow nested forms in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	if len(state.spec.NestedForms) > 0 {
		if _, ok := state.spec.NestedForms[current.Head.String()]; !ok {
			s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s cannot contain nested form %q", state.spec.Name, current.Head.String()))
			return
		}
	}
	nested := s.compileForm(current, locals)
	if nested != nil {
		state.form.Forms = append(state.form.Forms, *nested)
	}
}

func (s *compileState) execCall(state *formExecState, current *ast.CallStmt, locals *env) {
	if !allowsCall(state.spec.BodyMode) {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow call statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	call := Call{Name: current.Callee.String(), Pos: current.Pos(), End: current.End()}
	for _, arg := range current.Args {
		value, err := s.evalExpr(arg, locals)
		if err != nil {
			s.diags.AddError(arg.Pos(), arg.End(), err.Error())
			return
		}
		call.Args = append(call.Args, value)
	}
	spec, ok := s.compiler.actions[call.Name]
	if !ok {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("unknown action %q", call.Name))
		return
	}
	if err := validateArity("action", call.Name, spec.MinArgs, spec.MaxArgs, len(call.Args)); err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return
	}
	if spec.Validate != nil {
		if err := spec.Validate(call.Args); err != nil {
			s.diags.AddError(current.Pos(), current.End(), err.Error())
			return
		}
	}
	state.form.Calls = append(state.form.Calls, call)
}

func (s *compileState) execLocalBinding(state *formExecState, name string, typeExpr ast.TypeExpr, expr ast.Expr, locals *env) {
	if state.spec.BodyMode != schema.BodyScript {
		s.diags.AddError(expr.Pos(), expr.End(), fmt.Sprintf("%s does not allow script statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	value, err := s.evalExpr(expr, locals)
	if err != nil {
		s.diags.AddError(expr.Pos(), expr.End(), err.Error())
		return
	}
	if typeExpr != nil {
		if typ := s.convertType(typeExpr); typ != nil {
			if err := schema.CheckAssignable(typ, value); err != nil {
				s.diags.AddError(expr.Pos(), expr.End(), fmt.Sprintf("binding %q: %v", name, err))
				return
			}
		}
	}
	locals.Bind(name, value)
}

func (s *compileState) execIf(state *formExecState, current *ast.IfStmt, locals *env) {
	if state.spec.BodyMode != schema.BodyScript {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow script statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	value, err := s.evalExpr(current.Condition, locals)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return
	}
	cond, ok := value.(bool)
	if !ok {
		s.diags.AddError(current.Pos(), current.End(), "if condition must be bool")
		return
	}
	if cond {
		s.execFormBlock(state, current.Then, locals)
		return
	}
	if current.Else != nil {
		s.execFormBlock(state, current.Else, locals)
	}
}

func (s *compileState) execFor(state *formExecState, current *ast.ForStmt, locals *env) {
	if state.spec.BodyMode != schema.BodyScript {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow script statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	value, err := s.evalExpr(current.Iterable, locals)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return
	}
	items, err := iterateValues(value)
	if err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return
	}
	for _, item := range items {
		blockEnv := newEnv(locals)
		blockEnv.Bind(current.Name.Name, item)
		s.execFormBlock(state, current.Body, blockEnv)
	}
}

func (s *compileState) execFormBlock(state *formExecState, block *ast.Block, locals *env) {
	if block == nil {
		return
	}
	s.execFormItems(state, block.Items, newEnv(locals))
}

//nolint:nilnil // A missing return value from a user function is represented as nil.
func (s *compileState) callUserFunction(name string, decl *ast.FnDecl, args []any) (any, error) {
	if len(args) != len(decl.Params) {
		return nil, fmt.Errorf("function %q expects %d arguments", name, len(decl.Params))
	}

	locals := newEnv(nil)
	for idx, param := range decl.Params {
		value := args[idx]
		if param.Type != nil {
			if typ := s.convertType(param.Type); typ != nil {
				if err := schema.CheckAssignable(typ, value); err != nil {
					return nil, fmt.Errorf("function %q parameter %q: %w", name, param.Name.Name, err)
				}
			}
		}
		locals.Bind(param.Name.Name, value)
	}

	result, err := s.execFunctionBlock(decl.Body, locals)
	if err != nil {
		return nil, fmt.Errorf("function %q: %w", name, err)
	}
	if decl.Result != nil {
		if result.IsAbsent() {
			return nil, fmt.Errorf("function %q must return a value", name)
		}
		if typ := s.convertType(decl.Result); typ != nil {
			if err := schema.CheckAssignable(typ, result.MustGet()); err != nil {
				return nil, fmt.Errorf("function %q return type: %w", name, err)
			}
		}
	}
	if value, ok := result.Get(); ok {
		return value, nil
	}
	return nil, nil
}

func (s *compileState) execFunctionBlock(block *ast.Block, locals *env) (mo.Option[any], error) {
	if block == nil {
		return mo.None[any](), nil
	}
	for _, item := range block.Items {
		switch current := item.(type) {
		case *ast.ConstDecl:
			if err := s.bindFunctionLocal(current.Name.Name, current.Type, current.Value, locals); err != nil {
				return mo.None[any](), err
			}
		case *ast.LetDecl:
			if err := s.bindFunctionLocal(current.Name.Name, current.Type, current.Value, locals); err != nil {
				return mo.None[any](), err
			}
		case *ast.IfStmt:
			value, err := s.evalExpr(current.Condition, locals)
			if err != nil {
				return mo.None[any](), err
			}
			cond, ok := value.(bool)
			if !ok {
				return mo.None[any](), errors.New("if condition must be bool")
			}
			var branch *ast.Block
			if cond {
				branch = current.Then
			} else {
				branch = current.Else
			}
			result, err := s.execFunctionBlock(branch, newEnv(locals))
			if err != nil || result.IsPresent() {
				return result, err
			}
		case *ast.ForStmt:
			value, err := s.evalExpr(current.Iterable, locals)
			if err != nil {
				return mo.None[any](), err
			}
			items, err := iterateValues(value)
			if err != nil {
				return mo.None[any](), err
			}
			for _, itemValue := range items {
				blockEnv := newEnv(locals)
				blockEnv.Bind(current.Name.Name, itemValue)
				result, err := s.execFunctionBlock(current.Body, blockEnv)
				if err != nil || result.IsPresent() {
					return result, err
				}
			}
		case *ast.ReturnStmt:
			value, err := s.evalExpr(current.Value, locals)
			if err != nil {
				return mo.None[any](), err
			}
			return mo.Some[any](value), nil
		case *ast.ImportDecl:
			return mo.None[any](), errors.New("import is not allowed in function bodies")
		case *ast.FnDecl:
			return mo.None[any](), errors.New("nested function declarations are not implemented")
		case *ast.Assignment, *ast.CallStmt, *ast.FormDecl:
			return mo.None[any](), fmt.Errorf("unsupported function body item %T", current)
		default:
			return mo.None[any](), fmt.Errorf("unsupported function body item %T", current)
		}
	}
	return mo.None[any](), nil
}

func (s *compileState) bindFunctionLocal(name string, typeExpr ast.TypeExpr, expr ast.Expr, locals *env) error {
	value, err := s.evalExpr(expr, locals)
	if err != nil {
		return err
	}
	if typeExpr != nil {
		if typ := s.convertType(typeExpr); typ != nil {
			if err := schema.CheckAssignable(typ, value); err != nil {
				return fmt.Errorf("binding %q: %w", name, err)
			}
		}
	}
	locals.Bind(name, value)
	return nil
}

func iterateValues(value any) ([]any, error) {
	switch current := value.(type) {
	case []any:
		return current, nil
	case map[string]any:
		items := make([]any, 0, len(current))
		for _, item := range current {
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("for loop expects list or map, got %T", value)
	}
}
