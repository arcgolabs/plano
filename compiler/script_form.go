package compiler

import (
	"fmt"

	"github.com/arcgolabs/plano/ast"
	"github.com/arcgolabs/plano/schema"
)

type formExecState struct {
	spec      schema.FormSpec
	form      *Form
	hir       *HIRForm
	fieldSeen map[string]bool
}

func (s *compileState) execFormItems(state *formExecState, items []ast.FormItem, locals *env) {
	for _, item := range items {
		if s.execFormStructuralItem(state, item, locals) {
			continue
		}
		s.execUnsupportedFormItem(item)
	}
}

func (s *compileState) execFormStructuralItem(state *formExecState, item ast.FormItem, locals *env) bool {
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
	default:
		return false
	}
	return true
}

func (s *compileState) execUnsupportedFormItem(item ast.FormItem) {
	switch current := item.(type) {
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
	state.form.Fields.Set(current.Name.Name, value)
	state.fieldSeen[current.Name.Name] = true
	scopeID := locals.scope
	expected := fieldSpec.Type
	actual := staticTypeOfValue(value)
	if check, ok := s.fieldCheck(current.Pos(), current.End()); ok {
		scopeID = check.ScopeID
		expected = check.Expected
		actual = check.Actual
	}
	state.hir.Fields.Set(current.Name.Name, HIRField{
		Name:     fieldSpec.Name,
		ScopeID:  scopeID,
		Expected: expected,
		Actual:   actual,
		Value:    value,
		Pos:      current.Pos(),
		End:      current.End(),
	})
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
	nested, hirNested := s.compileForm(current, locals)
	if nested != nil && hirNested != nil {
		state.form.Forms = append(state.form.Forms, *nested)
		state.hir.Forms = append(state.hir.Forms, *hirNested)
	}
}

func (s *compileState) execCall(state *formExecState, current *ast.CallStmt, locals *env) {
	if !allowsCall(state.spec.BodyMode) {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("%s does not allow call statements in %s body", state.spec.Name, state.spec.BodyMode.String()))
		return
	}
	call, spec, ok := s.buildActionCall(current, locals)
	if !ok {
		return
	}
	state.form.Calls = append(state.form.Calls, call)
	state.hir.Calls = append(state.hir.Calls, s.lowerActionCall(current, locals.scope, call, spec))
}

func (s *compileState) buildActionCall(current *ast.CallStmt, locals *env) (Call, ActionSpec, bool) {
	call := Call{Name: current.Callee.String(), Pos: current.Pos(), End: current.End()}
	args, ok := s.evalCallStatementArgs(current, locals)
	if !ok {
		return Call{}, ActionSpec{}, false
	}
	call.Args = args
	spec, ok := s.compiler.actions.Get(call.Name)
	if !ok {
		s.diags.AddError(current.Pos(), current.End(), fmt.Sprintf("unknown action %q", call.Name))
		return Call{}, ActionSpec{}, false
	}
	if err := validateArity("action", call.Name, spec.MinArgs, spec.MaxArgs, len(call.Args)); err != nil {
		s.diags.AddError(current.Pos(), current.End(), err.Error())
		return Call{}, ActionSpec{}, false
	}
	if spec.Validate != nil {
		if err := spec.Validate(call.Args); err != nil {
			s.diags.AddError(current.Pos(), current.End(), err.Error())
			return Call{}, ActionSpec{}, false
		}
	}
	return call, spec, true
}

func (s *compileState) evalCallStatementArgs(current *ast.CallStmt, locals *env) ([]any, bool) {
	args := make([]any, 0, len(current.Args))
	for _, arg := range current.Args {
		value, err := s.evalExpr(arg, locals)
		if err != nil {
			s.diags.AddError(arg.Pos(), arg.End(), err.Error())
			return nil, false
		}
		args = append(args, value)
	}
	return args, true
}

func (s *compileState) lowerActionCall(current *ast.CallStmt, scopeID string, call Call, spec ActionSpec) HIRCall {
	var resultType schema.Type = schema.TypeAny
	if callCheck, ok := s.callCheck(current.Pos(), current.End()); ok {
		scopeID = callCheck.ScopeID
		resultType = callCheck.Result
	}
	hirArgs := make([]HIRArg, 0, len(call.Args))
	for idx, arg := range call.Args {
		hirArgs = append(hirArgs, HIRArg{
			Type:  signatureArgType(idx, spec.ArgTypes, spec.VariadicType),
			Value: arg,
		})
	}
	return HIRCall{
		Name:    call.Name,
		ScopeID: scopeID,
		Args:    hirArgs,
		Result:  resultType,
		Pos:     current.Pos(),
		End:     current.End(),
	}
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
		if typ := convertTypeExpr(typeExpr); typ != nil {
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
		blockEnv := s.newScopeEnv(locals, ScopeLoop, current.Pos(), current.End())
		blockEnv.Bind(current.Name.Name, item)
		s.execFormBlock(state, current.Body, blockEnv)
	}
}

func (s *compileState) execFormBlock(state *formExecState, block *ast.Block, locals *env) {
	if block == nil {
		return
	}
	s.execFormItems(state, block.Items, s.newScopeEnv(locals, ScopeBlock, block.Pos(), block.End()))
}
